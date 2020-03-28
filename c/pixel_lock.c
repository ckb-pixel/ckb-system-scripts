#include "blake2b.h"
#include "ckb_syscalls.h"
#include "common.h"
#include "protocol.h"
#include "secp256k1_helper.h"
#include "lock_utils.h"
#include "math.h"

#define BLAKE2B_BLOCK_SIZE 32
#define SCRIPT_SIZE 32768
#define CKB_LEN 8
#define MAX_WITNESS_SIZE 32768
#define MAX_TYPE_HASH 256

#define ERROR_ARGUMENTS_LEN -1
#define ERROR_ENCODING -2
#define ERROR_SYSCALL -3
#define ERROR_SCRIPT_TOO_LONG -21
#define ERROR_OUTPUT_AMOUNT_NOT_ENOUGH -52
#define ERROR_TOO_MUCH_TYPE_HASH_INPUTS -53
#define ERROR_PARING_INPUT_FAILED -54
#define ERROR_PARING_OUTPUT_FAILED -55
#define ERROR_DUPLICATED_INPUT_TYPE_HASH -56
#define ERROR_DUPLICATED_OUTPUT_TYPE_HASH -57
#define ERROR_OFFICIAL_FEE -58

typedef struct {
  unsigned char type_hash[BLAKE2B_BLOCK_SIZE];
  uint64_t ckb_amount;
  uint32_t output_cnt;
} InputWallet;

int uint64_overflow_add1(uint64_t * result, uint64_t a){
  *result = a + a * 2 / 10;
  return 0;
}

int check_payment_unlock() {
  unsigned char lock_hash[BLAKE2B_BLOCK_SIZE];
  InputWallet input_wallets[MAX_TYPE_HASH];
  uint64_t len = BLAKE2B_BLOCK_SIZE;
  /* load wallet lock hash */
  int ret = ckb_load_script_hash(lock_hash, &len, 0);
  if (ret != CKB_SUCCESS) {
    return ERROR_SYSCALL;
  }
  if (len > BLAKE2B_BLOCK_SIZE) {
    return ERROR_SCRIPT_TOO_LONG;
  }

  /* iterate inputs and find input wallet cell */
  uint64_t total_input_ckb = 0;
  int i = 0;
  len = BLAKE2B_BLOCK_SIZE;
  while (1) {
    if (i >= MAX_TYPE_HASH) {
      return ERROR_TOO_MUCH_TYPE_HASH_INPUTS;
    }

    ret = ckb_load_cell_by_field(input_wallets[i].type_hash, &len, 0, i,
                                         CKB_SOURCE_GROUP_INPUT,
                                         CKB_CELL_FIELD_TYPE_HASH);

    if (ret == CKB_INDEX_OUT_OF_BOUND) {
      break;
    }

    if (ret != CKB_SUCCESS) {
      return ERROR_SYSCALL;
    }

    if (len != BLAKE2B_BLOCK_SIZE) {
      return ERROR_ENCODING;
    }

    /* load amount */
    len = CKB_LEN;
    ret = ckb_load_cell_by_field(
        (uint8_t *)&input_wallets[i].ckb_amount, &len, 0, i,
        CKB_SOURCE_GROUP_INPUT, CKB_CELL_FIELD_CAPACITY);
    if (ret != CKB_SUCCESS) {
      return ERROR_SYSCALL;
    }
    if (len != CKB_LEN) {
      return ERROR_ENCODING;
    }

    total_input_ckb += input_wallets[i].ckb_amount;

    i++;
  }

  int input_wallets_cnt = i;

  /* iterate outputs wallet cell */
  i = 0;
  while (1) {
    uint8_t output_lock_hash[BLAKE2B_BLOCK_SIZE];
    uint64_t len = BLAKE2B_BLOCK_SIZE;
    /* check lock hash */
    ret = ckb_load_cell_by_field(output_lock_hash, &len, 0, i,
                                         CKB_SOURCE_OUTPUT,
                                         CKB_CELL_FIELD_LOCK_HASH);
    if (ret == CKB_INDEX_OUT_OF_BOUND) {
      break;
    }
    if (ret != CKB_SUCCESS) {
      return ret;
    }
    if (len != BLAKE2B_BLOCK_SIZE) {
      return ERROR_ENCODING;
    }
    int has_same_lock =
        memcmp(output_lock_hash, lock_hash, BLAKE2B_BLOCK_SIZE) == 0;
    if (!has_same_lock) {
      i++;
      continue;
    }

    /* load amount */
    uint64_t ckb_amount;
    len = CKB_LEN;
    ret = ckb_load_cell_by_field((uint8_t *)&ckb_amount, &len, 0, i,
                                         CKB_SOURCE_OUTPUT,
                                         CKB_CELL_FIELD_CAPACITY);
    if (ret != CKB_SUCCESS) {
      return ERROR_SYSCALL;
    }
    if (len != CKB_LEN) {
      return ERROR_ENCODING;
    }

    /* find input wallet which has same type hash */
    int found_inputs = 0;
    for (int j = 0; j < input_wallets_cnt; j++) {
      /* compare amount */
      uint64_t min_output_ckb_amount;
      int overflow;
      overflow = uint64_overflow_add1(&min_output_ckb_amount, input_wallets[j].ckb_amount);
      int invalid_output_ckb = overflow || ckb_amount < min_output_ckb_amount;

      /* fail the unlock if both conditions can't satisfied */
      if(invalid_output_ckb) {
        return ERROR_OUTPUT_AMOUNT_NOT_ENOUGH;
      }

      /* increase counter */
      found_inputs++;
      input_wallets[j].output_cnt += 1;
      if (found_inputs > 1) {
        return ERROR_DUPLICATED_INPUT_TYPE_HASH;
      }
      if (input_wallets[j].output_cnt > 1) {
        return ERROR_DUPLICATED_OUTPUT_TYPE_HASH;
      }
    }

    /* one output should pair with one input */
    if (found_inputs == 0) {
      return ERROR_PARING_OUTPUT_FAILED;
    } else if (found_inputs > 1) {
      return ERROR_DUPLICATED_INPUT_TYPE_HASH;
    }

    i++;
  }

  /* check inputs wallet, one input should pair with one output */
  for (int j = 0; j < input_wallets_cnt; j++) {
    if (input_wallets[j].output_cnt == 0) {
      return ERROR_PARING_INPUT_FAILED;
    } else if (input_wallets[j].output_cnt > 1) {
      return ERROR_DUPLICATED_OUTPUT_TYPE_HASH;
    }
  }

  /* check official lock */
  uint64_t total_output_official_ckb = 0;
  uint8_t official_lock_hash[BLAKE2B_BLOCK_SIZE] = {
    106, 36, 43, 87, 34, 116, 132, 233, 4, 180, 224, 139, 169, 111, 25, 166, 35, 195, 103, 220, 189, 24, 103, 94, 198, 242, 167, 26, 15, 244, 236, 38
  };
  i = 0;
  while (1) {
    uint8_t output_lock_hash[BLAKE2B_BLOCK_SIZE];
    uint64_t len = BLAKE2B_BLOCK_SIZE;
    /* check lock hash */
    ret = ckb_load_cell_by_field(output_lock_hash, &len, 0, i,
                                         CKB_SOURCE_OUTPUT,
                                         CKB_CELL_FIELD_LOCK_HASH);
    if (ret == CKB_INDEX_OUT_OF_BOUND) {
      break;
    }
    if (ret != CKB_SUCCESS) {
      return ret;
    }
    if (len != BLAKE2B_BLOCK_SIZE) {
      return ERROR_ENCODING;
    }
    int has_same_lock =
        memcmp(output_lock_hash, official_lock_hash, BLAKE2B_BLOCK_SIZE) == 0;
    if (!has_same_lock) {
      i++;
      continue;
    }

    /* load amount */
    uint64_t ckb_amount;
    len = CKB_LEN;
    ret = ckb_load_cell_by_field((uint8_t *)&ckb_amount, &len, 0, i,
                                         CKB_SOURCE_OUTPUT,
                                         CKB_CELL_FIELD_CAPACITY);
    if (ret != CKB_SUCCESS) {
      return ERROR_SYSCALL;
    }
    if (len != CKB_LEN) {
      return ERROR_ENCODING;
    }

    total_output_official_ckb += ckb_amount;

    i++;
  }

  if (total_output_official_ckb < total_input_ckb / 10) {
    return ERROR_OFFICIAL_FEE;
  }

  return CKB_SUCCESS;
}

int has_signature(int *has_sig) {
  int ret;
  unsigned char temp[MAX_WITNESS_SIZE];

  /* Load witness of first input */
  uint64_t witness_len = MAX_WITNESS_SIZE;
  ret = ckb_load_witness(temp, &witness_len, 0, 0, CKB_SOURCE_GROUP_INPUT);

  if ((ret == CKB_INDEX_OUT_OF_BOUND) || (ret == CKB_SUCCESS && witness_len == 0)) {
    *has_sig = 0;
    return CKB_SUCCESS;
  }

  if (ret != CKB_SUCCESS) {
    return ERROR_SYSCALL;
  }

  if (witness_len > MAX_WITNESS_SIZE) {
    return ERROR_WITNESS_SIZE;
  }

  /* load signature */
  mol_seg_t lock_bytes_seg;
  ret = extract_witness_lock(temp, witness_len, &lock_bytes_seg);
  if (ret != 0) {
    return ERROR_ENCODING;
  }

  *has_sig = lock_bytes_seg.size > 0;
  return CKB_SUCCESS;
}

int read_args(unsigned char *pubkey_hash) {
  int ret;
  uint64_t len = 0;

  /* Load args */
  unsigned char script[SCRIPT_SIZE];
  len = SCRIPT_SIZE;
  ret = ckb_load_script(script, &len, 0);
  if (ret != CKB_SUCCESS) {
    return ERROR_SYSCALL;
  }
  if (len > SCRIPT_SIZE) {
    return ERROR_SCRIPT_TOO_LONG;
  }
  mol_seg_t script_seg;
  script_seg.ptr = (uint8_t *)script;
  script_seg.size = len;

  if (MolReader_Script_verify(&script_seg, false) != MOL_OK) {
    return ERROR_ENCODING;
  }

  mol_seg_t args_seg = MolReader_Script_get_args(&script_seg);
  mol_seg_t args_bytes_seg = MolReader_Bytes_raw_bytes(&args_seg);
  if (args_bytes_seg.size != BLAKE160_SIZE) {
    return ERROR_ARGUMENTS_LEN;
  }
  memcpy(pubkey_hash, args_bytes_seg.ptr, BLAKE160_SIZE);
  return CKB_SUCCESS;
}

int main() {
  int ret;
  int has_sig;
  unsigned char pubkey_hash[BLAKE160_SIZE];
  ret = read_args(pubkey_hash);
  if (ret != CKB_SUCCESS) {
    return ret;
  }
  ret = has_signature(&has_sig);
  if (ret != CKB_SUCCESS) {
    return ret;
  }
  if (has_sig) {
    /* unlock via signature */
    return verify_secp256k1_blake160_sighash_all(pubkey_hash);
  } else {
    /* unlock via payment */
    return check_payment_unlock();
  }
}