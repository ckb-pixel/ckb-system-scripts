#include "protocol.h"
#include "ckb_syscalls.h"

#define BLAKE2B_BLOCK_SIZE 32
#define SCRIPT_SIZE 32768

#define ERROR_ARGUMENTS_LEN -1
#define ERROR_ENCODING -2
#define ERROR_SYSCALL -3
#define ERROR_SCRIPT_TOO_LONG -21
#define ERROR_OVERFLOWING -51
#define ERROR_COORDINATE -61

int main() {
  unsigned char script[SCRIPT_SIZE];
  uint64_t len = SCRIPT_SIZE;
  int ret = ckb_load_script(script, &len, 0);
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
  if (args_bytes_seg.size != BLAKE2B_BLOCK_SIZE) {
    return ERROR_ARGUMENTS_LEN;
  }

  int owner_mode = 0;
  size_t i = 0;
  while (1) {
    uint8_t buffer[BLAKE2B_BLOCK_SIZE];
    uint64_t len = BLAKE2B_BLOCK_SIZE;
    ret = ckb_load_cell_by_field(buffer, &len, 0, i, CKB_SOURCE_INPUT,
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
    if (memcmp(buffer, args_bytes_seg.ptr, BLAKE2B_BLOCK_SIZE) == 0) {
      owner_mode = 1;
      break;
    }
    i += 1;
  }

  if (owner_mode) {
    return CKB_SUCCESS;
  }

  int r = -61;
  i = 0;
  while (1) {
    uint8_t current_data[5];
    len = 5;
    ret = ckb_load_cell_data((uint8_t *)&current_data, &len, 0, i,
                             CKB_SOURCE_GROUP_OUTPUT);
    if (ret == CKB_INDEX_OUT_OF_BOUND) {
      break;
    }
    if (ret != CKB_SUCCESS) {
      return ret;
    }
    if (len != 5) {
      return ERROR_ENCODING;
    }

    size_t j = 0;
    while (1) {
      uint8_t current_data_i[5];
      uint64_t len_i = 5;
      int ret_i = ckb_load_cell_data((uint8_t *)&current_data_i, &len_i, 0, j,
                               CKB_SOURCE_GROUP_INPUT);
      if (ret_i == CKB_INDEX_OUT_OF_BOUND) {
        break;
      }
      if (ret_i != CKB_SUCCESS) {
        return ret;
      }
      if (len_i != 5) {
        return ERROR_ENCODING;
      }

      if (current_data[0] == current_data_i[0] && current_data[1] == current_data_i[1]) {
        r = 0;
        break;
      }
      j += 1;
    }

    if (r != 0) {
      return ERROR_COORDINATE;
    }

    i += 1;
  }
  return CKB_SUCCESS;
}