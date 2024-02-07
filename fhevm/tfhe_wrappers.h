#include <tfhe.h>

#undef NDEBUG
#include <assert.h>

typedef struct FhevmKeys{
	void *sks, *cks, *pks;
} FhevmKeys;

FhevmKeys generate_fhevm_keys();

int serialize_compact_public_key(void *pks, DynamicBuffer* out);

void* deserialize_server_key(DynamicBufferView in);

void* deserialize_client_key(DynamicBufferView in);

void* deserialize_compact_public_key(DynamicBufferView in);

void checked_set_server_key(void *sks);

void* cast_8_bool(void* ct, void* sks);

void* cast_bool_8(void* ct, void* sks);

void* cast_bool_16(void* ct, void* sks);

void* cast_bool_32(void* ct, void* sks);

void* cast_bool_64(void* ct, void* sks);

int serialize_fhe_uint8(void *ct, DynamicBuffer* out);

void* deserialize_fhe_uint8(DynamicBufferView in);

void* deserialize_compact_fhe_uint8(DynamicBufferView in);

int serialize_fhe_uint16(void *ct, DynamicBuffer* out);

void* deserialize_fhe_uint16(DynamicBufferView in);

void* deserialize_compact_fhe_uint16(DynamicBufferView in);

int serialize_fhe_uint32(void *ct, DynamicBuffer* out);

void* deserialize_fhe_uint32(DynamicBufferView in);

void* deserialize_compact_fhe_uint32(DynamicBufferView in);

int serialize_fhe_uint64(void *ct, DynamicBuffer* out);

void* deserialize_fhe_uint64(DynamicBufferView in);

void* deserialize_compact_fhe_uint64(DynamicBufferView in);

void destroy_fhe_uint8(void* ct);

void destroy_fhe_uint16(void* ct);

void destroy_fhe_uint32(void* ct);

void destroy_fhe_uint64(void* ct);

void* add_fhe_uint8(void* ct1, void* ct2, void* sks);

void* add_fhe_uint16(void* ct1, void* ct2, void* sks);

void* add_fhe_uint32(void* ct1, void* ct2, void* sks);

void* add_fhe_uint64(void* ct1, void* ct2, void* sks);

void* scalar_add_fhe_uint8(void* ct, uint8_t pt, void* sks);

void* scalar_add_fhe_uint16(void* ct, uint16_t pt, void* sks);

void* scalar_add_fhe_uint32(void* ct, uint32_t pt, void* sks);

void* scalar_add_fhe_uint64(void* ct, uint64_t pt, void* sks);

void* sub_fhe_uint8(void* ct1, void* ct2, void* sks);

void* sub_fhe_uint16(void* ct1, void* ct2, void* sks);

void* sub_fhe_uint32(void* ct1, void* ct2, void* sks);

void* sub_fhe_uint64(void* ct1, void* ct2, void* sks);

void* scalar_sub_fhe_uint8(void* ct, uint8_t pt, void* sks);

void* scalar_sub_fhe_uint16(void* ct, uint16_t pt, void* sks);

void* scalar_sub_fhe_uint32(void* ct, uint32_t pt, void* sks);

void* scalar_sub_fhe_uint64(void* ct, uint64_t pt, void* sks);

void* mul_fhe_uint8(void* ct1, void* ct2, void* sks);

void* mul_fhe_uint16(void* ct1, void* ct2, void* sks);

void* mul_fhe_uint32(void* ct1, void* ct2, void* sks);

void* mul_fhe_uint64(void* ct1, void* ct2, void* sks);

void* scalar_mul_fhe_uint8(void* ct, uint8_t pt, void* sks);

void* scalar_mul_fhe_uint16(void* ct, uint16_t pt, void* sks);

void* scalar_mul_fhe_uint32(void* ct, uint32_t pt, void* sks);

void* scalar_mul_fhe_uint64(void* ct, uint64_t pt, void* sks);

void* scalar_div_fhe_uint8(void* ct, uint8_t pt, void* sks);

void* scalar_div_fhe_uint16(void* ct, uint16_t pt, void* sks);

void* scalar_div_fhe_uint32(void* ct, uint32_t pt, void* sks);

void* scalar_div_fhe_uint64(void* ct, uint64_t pt, void* sks);

void* scalar_rem_fhe_uint8(void* ct, uint8_t pt, void* sks);

void* scalar_rem_fhe_uint16(void* ct, uint16_t pt, void* sks);

void* scalar_rem_fhe_uint32(void* ct, uint32_t pt, void* sks);

void* scalar_rem_fhe_uint64(void* ct, uint64_t pt, void* sks);

void* bitand_fhe_uint8(void* ct1, void* ct2, void* sks);

void* bitand_fhe_uint16(void* ct1, void* ct2, void* sks);

void* bitand_fhe_uint32(void* ct1, void* ct2, void* sks);

void* bitand_fhe_uint64(void* ct1, void* ct2, void* sks);

void* bitor_fhe_uint8(void* ct1, void* ct2, void* sks);

void* bitor_fhe_uint16(void* ct1, void* ct2, void* sks);

void* bitor_fhe_uint32(void* ct1, void* ct2, void* sks);

void* bitor_fhe_uint64(void* ct1, void* ct2, void* sks);

void* bitxor_fhe_uint8(void* ct1, void* ct2, void* sks);

void* bitxor_fhe_uint16(void* ct1, void* ct2, void* sks);

void* bitxor_fhe_uint32(void* ct1, void* ct2, void* sks);

void* bitxor_fhe_uint64(void* ct1, void* ct2, void* sks);

void* shl_fhe_uint8(void* ct1, void* ct2, void* sks);

void* shl_fhe_uint16(void* ct1, void* ct2, void* sks);

void* shl_fhe_uint32(void* ct1, void* ct2, void* sks);

void* shl_fhe_uint64(void* ct1, void* ct2, void* sks);

void* scalar_shl_fhe_uint8(void* ct, uint8_t pt, void* sks);

void* scalar_shl_fhe_uint16(void* ct, uint16_t pt, void* sks);

void* scalar_shl_fhe_uint32(void* ct, uint32_t pt, void* sks);

void* scalar_shl_fhe_uint64(void* ct, uint64_t pt, void* sks);

void* shr_fhe_uint8(void* ct1, void* ct2, void* sks);

void* shr_fhe_uint16(void* ct1, void* ct2, void* sks);

void* shr_fhe_uint32(void* ct1, void* ct2, void* sks);

void* shr_fhe_uint64(void* ct1, void* ct2, void* sks);

void* scalar_shr_fhe_uint8(void* ct, uint8_t pt, void* sks);

void* scalar_shr_fhe_uint16(void* ct, uint16_t pt, void* sks);

void* scalar_shr_fhe_uint32(void* ct, uint32_t pt, void* sks);

void* scalar_shr_fhe_uint64(void* ct, uint64_t pt, void* sks);

void* eq_fhe_uint8(void* ct1, void* ct2, void* sks);

void* eq_fhe_uint16(void* ct1, void* ct2, void* sks);

void* eq_fhe_uint32(void* ct1, void* ct2, void* sks);

void* eq_fhe_uint64(void* ct1, void* ct2, void* sks);

void* scalar_eq_fhe_uint8(void* ct, uint8_t pt, void* sks);

void* scalar_eq_fhe_uint16(void* ct, uint16_t pt, void* sks);

void* scalar_eq_fhe_uint32(void* ct, uint32_t pt, void* sks);

void* scalar_eq_fhe_uint64(void* ct, uint64_t pt, void* sks);

void* ne_fhe_uint8(void* ct1, void* ct2, void* sks);

void* ne_fhe_uint16(void* ct1, void* ct2, void* sks);

void* ne_fhe_uint32(void* ct1, void* ct2, void* sks);

void* ne_fhe_uint64(void* ct1, void* ct2, void* sks);

void* scalar_ne_fhe_uint8(void* ct, uint8_t pt, void* sks);

void* scalar_ne_fhe_uint16(void* ct, uint16_t pt, void* sks);

void* scalar_ne_fhe_uint32(void* ct, uint32_t pt, void* sks);

void* scalar_ne_fhe_uint64(void* ct, uint64_t pt, void* sks);

void* ge_fhe_uint8(void* ct1, void* ct2, void* sks);

void* ge_fhe_uint16(void* ct1, void* ct2, void* sks);

void* ge_fhe_uint32(void* ct1, void* ct2, void* sks);

void* ge_fhe_uint64(void* ct1, void* ct2, void* sks);

void* scalar_ge_fhe_uint8(void* ct, uint8_t pt, void* sks);

void* scalar_ge_fhe_uint16(void* ct, uint16_t pt, void* sks);

void* scalar_ge_fhe_uint32(void* ct, uint32_t pt, void* sks);

void* scalar_ge_fhe_uint64(void* ct, uint64_t pt, void* sks);

void* gt_fhe_uint8(void* ct1, void* ct2, void* sks);

void* gt_fhe_uint16(void* ct1, void* ct2, void* sks);

void* gt_fhe_uint32(void* ct1, void* ct2, void* sks);

void* gt_fhe_uint64(void* ct1, void* ct2, void* sks);

void* scalar_gt_fhe_uint8(void* ct, uint8_t pt, void* sks);

void* scalar_gt_fhe_uint16(void* ct, uint16_t pt, void* sks);

void* scalar_gt_fhe_uint32(void* ct, uint32_t pt, void* sks);

void* scalar_gt_fhe_uint64(void* ct, uint64_t pt, void* sks);

void* le_fhe_uint8(void* ct1, void* ct2, void* sks);

void* le_fhe_uint16(void* ct1, void* ct2, void* sks);

void* le_fhe_uint32(void* ct1, void* ct2, void* sks);

void* le_fhe_uint64(void* ct1, void* ct2, void* sks);

void* scalar_le_fhe_uint8(void* ct, uint8_t pt, void* sks);

void* scalar_le_fhe_uint16(void* ct, uint16_t pt, void* sks);

void* scalar_le_fhe_uint32(void* ct, uint32_t pt, void* sks);

void* scalar_le_fhe_uint64(void* ct, uint64_t pt, void* sks);

void* lt_fhe_uint8(void* ct1, void* ct2, void* sks);

void* lt_fhe_uint16(void* ct1, void* ct2, void* sks);

void* lt_fhe_uint32(void* ct1, void* ct2, void* sks);

void* lt_fhe_uint64(void* ct1, void* ct2, void* sks);

void* scalar_lt_fhe_uint8(void* ct, uint8_t pt, void* sks);

void* scalar_lt_fhe_uint16(void* ct, uint16_t pt, void* sks);

void* scalar_lt_fhe_uint32(void* ct, uint32_t pt, void* sks);

void* scalar_lt_fhe_uint64(void* ct, uint64_t pt, void* sks);

void* min_fhe_uint8(void* ct1, void* ct2, void* sks);

void* min_fhe_uint16(void* ct1, void* ct2, void* sks);

void* min_fhe_uint32(void* ct1, void* ct2, void* sks);

void* min_fhe_uint64(void* ct1, void* ct2, void* sks);

void* scalar_min_fhe_uint8(void* ct, uint8_t pt, void* sks);

void* scalar_min_fhe_uint16(void* ct, uint16_t pt, void* sks);

void* scalar_min_fhe_uint32(void* ct, uint32_t pt, void* sks);

void* scalar_min_fhe_uint64(void* ct, uint64_t pt, void* sks);

void* max_fhe_uint8(void* ct1, void* ct2, void* sks);

void* max_fhe_uint16(void* ct1, void* ct2, void* sks);

void* max_fhe_uint32(void* ct1, void* ct2, void* sks);

void* max_fhe_uint64(void* ct1, void* ct2, void* sks);

void* scalar_max_fhe_uint8(void* ct, uint8_t pt, void* sks);

void* scalar_max_fhe_uint16(void* ct, uint16_t pt, void* sks);

void* scalar_max_fhe_uint32(void* ct, uint32_t pt, void* sks);

void* scalar_max_fhe_uint64(void* ct, uint64_t pt, void* sks);

void* neg_fhe_uint8(void* ct, void* sks);

void* neg_fhe_uint16(void* ct, void* sks);

void* neg_fhe_uint32(void* ct, void* sks);

void* neg_fhe_uint64(void* ct, void* sks);

void* not_fhe_uint8(void* ct, void* sks);

void* not_fhe_uint16(void* ct, void* sks);

void* not_fhe_uint32(void* ct, void* sks);

void* not_fhe_uint64(void* ct, void* sks);

void* if_then_else_fhe_uint8(void* condition, void* ct1, void* ct2, void* sks);

void* if_then_else_fhe_uint16(void* condition, void* ct1, void* ct2, void* sks);

void* if_then_else_fhe_uint32(void* condition, void* ct1, void* ct2, void* sks);

void* if_then_else_fhe_uint64(void* condition, void* ct1, void* ct2, void* sks);

int decrypt_fhe_uint8(void* cks, void* ct, uint8_t* res);

int decrypt_fhe_uint16(void* cks, void* ct, uint16_t* res);

int decrypt_fhe_uint32(void* cks, void* ct, uint32_t* res);

int decrypt_fhe_uint64(void* cks, void* ct, uint64_t* res);

void* public_key_encrypt_fhe_uint8(void* pks, uint8_t value);

void* public_key_encrypt_fhe_uint16(void* pks, uint16_t value);

void* public_key_encrypt_fhe_uint32(void* pks, uint32_t value);

void* public_key_encrypt_fhe_uint64(void* pks, uint64_t value);

void* trivial_encrypt_fhe_uint8(void* sks, uint8_t value);

void* trivial_encrypt_fhe_uint16(void* sks, uint16_t value);

void* trivial_encrypt_fhe_uint32(void* sks, uint32_t value);

void* trivial_encrypt_fhe_uint64(void* sks, uint64_t value);

void public_key_encrypt_and_serialize_fhe_uint8_list(void* pks, uint8_t value, DynamicBuffer* out);

void public_key_encrypt_and_serialize_fhe_uint16_list(void* pks, uint16_t value, DynamicBuffer* out);

void public_key_encrypt_and_serialize_fhe_uint32_list(void* pks, uint32_t value, DynamicBuffer* out);

void public_key_encrypt_and_serialize_fhe_uint64_list(void* pks, uint64_t value, DynamicBuffer* out);

void* cast_8_16(void* ct, void* sks);

void* cast_8_32(void* ct, void* sks);

void* cast_8_64(void* ct, void* sks);

void* cast_16_8(void* ct, void* sks);

void* cast_16_32(void* ct, void* sks);

void* cast_16_64(void* ct, void* sks);

void* cast_32_8(void* ct, void* sks);

void* cast_32_16(void* ct, void* sks);

void* cast_32_64(void* ct, void* sks);

void* cast_64_8(void* ct, void* sks);

void* cast_64_16(void* ct, void* sks);

void* cast_64_32(void* ct, void* sks);
