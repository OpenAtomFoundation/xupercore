// 创建bls账户
char* create_new_bls_account();

// 计算公共公钥
char* sum_bls_public_key(char *str_bls_public_key_array);

// 计算偏离系数K
char* get_bls_k(char *str_bls_public_key, char *str_bls_public_key_sum);

// 计算对应的公钥碎片
char* get_bls_public_key_part(char *str_bls_public_key, char *str_k);

// 计算签名碎片
char* get_bls_m(char *str_k, char *str_bls_private_key, char *str_index, char *str_bls_public_key);

// 计算签名碎片集合MK(i)
char* get_bls_mk(char *str_m_array);

// 验证签名碎片集合MK(i)的正确性
bool verify_bls_mk(char *str_bls_public_key, char *str_index, char *str_mk);

// 各个节点计算出自己的BLS签名片段
char* bls_sign(char *str_private_info, char *str_msg);

// 组合BLS签名片段，生成最终签名
char* bls_combine_sign(char *str_sign_array);

// 验证门限签名正确性
bool bls_verify_sign(char *str_bls_threshold_public_key, char *str_threshold_sig, char *str_msg);
