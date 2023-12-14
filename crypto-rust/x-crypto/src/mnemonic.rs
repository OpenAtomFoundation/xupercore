use std::collections::HashMap;
//use sha256::digest_bytes;
use crypto::sha2::Sha256;
use crypto::digest::Digest;

use crate::my_error::{ErrorKind,CryptoError,CryptoResult};
use crate::wordlist;
use crate::utils;

use num_bigint::{BigInt,ToBigInt,Sign};
use num_traits::Zero;

// struct 继承clone后，可以直接返回值了，从而解决生命周期的问题

#[allow(dead_code)]
/// 根据参数中提供的熵来生成一串助记词，参数中的熵应该是调用GenerateEntropy函数生成的熵。
pub fn generate_mnemonic_sentence_from_entropy(entropy: Vec<u8>, language_type: wordlist::LanguageType) -> CryptoResult<String> {
    // 先获得参数中熵对应的比特位长度，1个字节=8个比特
    let entropy_bit_length = entropy.len() * 8;
    
//    println!("entropy within fn[generate_mnemonic_sentence_from_entropy] is: {:?}", entropy);
    
    // 万一有人不按照函数说明先调用GenerateEntropy函数来生成熵呢？
    // 拖出去TJJTDS
    // 这里还要再校验一遍熵的长度是否符合规范
    validate_entropy_bit_size(entropy_bit_length)?;
    
    // 根据指定的语言类型来选择助记词词库
    let mut new_wordlist = wordlist::Wordlist {
        language: language_type,
        word_list: Vec::new(),
        word_map: HashMap::new(),
    };
    
    wordlist::GetWord::get_wordlist(&mut new_wordlist)?;
    
    // 再根据熵的比特位长度来计算其校验值所需要的比特位长度
    let checksum_bit_length = entropy_bit_length / 32;

    // 然后计算拼接后的字符串能转换为多少个助记词
    // 注意：每11个比特位对应一个数字，数字范围是0-2047，数字会再转换为对应的助记词
    let sentence_length = (entropy_bit_length + checksum_bit_length) / 11;

    // 熵的后面带上一段校验位
    let entropy_with_checksum = add_checksum(entropy);
    
    // 把熵切分为11个比特长度的片段
    // 把最右侧的11个比特长度的片段转化为数字，再匹配到对应的助记词
    // 然后再右移11个比特，再把最右侧的11个比特长度的片段转化为数字，再匹配到对应的助记词
    // 重复以上过程，直到熵被全部处理完成

    // 把带有校验值的熵转化为一个bigint，方便后续做比特位运算（主要是移位操作）
    // 将一个 big endian formatted u8 vec 转换为一个 big int
    let mut entropy_bigint =  BigInt::from_bytes_be(Sign::Plus, &entropy_with_checksum);

    // 创建一个string slice来为保存助记词做准备
    let mut words: Vec<String> = Vec::new();
    
    // 创建一个比特位全是0的空词，为后面通过比特位“And与”运算来获取熵的11个比特长度的片段做准备
    let mut word: BigInt;
    
    // 11111111111 - 11个1，当一个大的bigint和它进行“And”比特运算的时候，就会获得大的bigint最右边11位的比特位
    let last11_bits_mask = BigInt::from(2047);
    
    // 100000000000 - 除以这个带有11个0的数等于右移11个比特位
    let right_shift_11_bits_divider = BigInt::from(2048);
    
    // 填充助记词slice
    let mut index: usize = 0;
//    let zero_usize: usize = 0;
    while index < sentence_length {
        // 获取最右边的11个比特
        word = entropy_bigint.clone() & &last11_bits_mask;
        
        // 将熵右移11个比特
        entropy_bigint = entropy_bigint / &right_shift_11_bits_divider;
        
        // 把11个比特补齐为 2个字节
        let (_, word_bytes) = word.to_bytes_be();
        
        let word_bytes = utils::bytes::bytes_pad(word_bytes, 2);
        
        // vec转定长array
        // If the length doesn’t match, the input comes back in Err, and the program will panic,
        // but it won't happen!
        let word_bytes_array: [u8; 2] = word_bytes.try_into().unwrap();

        // 将2个字节编码为Uint16格式，然后在word list里面寻找对应的助记词
        let word_index = u16::from_be_bytes(word_bytes_array);
        
        let word_index_usize = usize::from(word_index);
        
        words.push(new_wordlist.word_list[word_index_usize].clone());
        
        index += 1;
    }
    
    words.reverse();
    
    let mnemonic_sentence = words.join(" ");

    Ok(mnemonic_sentence)
}

///  检查试图获取的Entropy的比特大小是否符合规范要求：
//  在128-256之间，并且是32的倍数
//  为什么这么设计，详见比特币改进计划第39号提案的数学模型
//
//  checksum length (CS)
//  entropy length (ENT)
//  mnemonic sentence (MS)
//
//  CS = ENT / 32
//  MS = (ENT + CS) / 11
//
//  |  ENT  | CS | ENT+CS |  MS  |
//  +-------+----+--------+------+
//  |  128  |  4 |   132  |  12  |
//  |  160  |  5 |   165  |  15  |
//  |  192  |  6 |   198  |  18  |
//  |  224  |  7 |   231  |  21  |
//  |  256  |  8 |   264  |  24  |
pub fn validate_entropy_bit_size(bit_size: usize) -> CryptoResult<bool> {
    if (bit_size%32) != 0 || bit_size < 128 || bit_size > 256 {
        Err(
            CryptoError {
                kind: ErrorKind::ErrInvalidEntropyLength,
                message: ErrorKind::ErrInvalidEntropyLength.to_string(),
            }
        )
    } else {
        Ok(true)
    }
    
}

/// 检查原始随机熵的字节长度是否符合约束性要求
/// +8的原因在于引入了8个bit的标记位来定义使用的密码学算法
pub fn validate_raw_entropy_bit_size(bit_size: usize) -> CryptoResult<bool> {
    if ((bit_size+8)%32) != 0 || (bit_size+8) < 128 || (bit_size+8) > 256 {
        Err(
            CryptoError {
                kind: ErrorKind::ErrInvalidRawEntropyLength,
                message: ErrorKind::ErrInvalidRawEntropyLength.to_string(),
            }
        )
    } else {
        Ok(true)
    }
    
}

#[allow(dead_code)]
/// 从助记词中提取出随机熵
pub fn get_entropy_from_mnemonic_sentence(mnemonic: String, language_type: wordlist::LanguageType) -> CryptoResult<Vec<u8>> {
    // 先判断助记词是否合法，也就是判断是否每个词都存在于助记词列表中
    let words_result = get_words_from_valid_mnemonic_sentence(mnemonic, language_type);
    
    // 判断是否从助记词字符串中成功的取出了通过合法性检查的助记词
    let words = match words_result {
        Ok(wordlist) => wordlist,
        Err(error) => return Err(error),
    };
    
    // 再判断助记词的校验位是否合法
    // 每个词的信息长度为11
    let mnemonic_bit_size = words.len() * 11;
    
    // 进一步计算出校验位的比特位长度
    let checksum_bit_size = mnemonic_bit_size % 32;
    
    let mut b: BigInt = Zero::zero();    
    
    // 根据语言加载对应的反向助记词map
    let mut new_wordlist = wordlist::Wordlist {
        language: language_type,
        word_list: Vec::new(),
        word_map: HashMap::new(),
    };
    
    wordlist::GetWord::get_wordlist(&mut new_wordlist)?;
    wordlist::GetWord::get_reversed_wordmap(&mut new_wordlist)?;
        
    let reversed_wordmap = new_wordlist.word_map;
    
    // 100000000000 - 除以这个带有11个0的数等于右移11个比特位
    let right_shift_11_bits_divider = BigInt::from(2048);
    
    for word in &words {
        let index = reversed_wordmap[word];

        // 将一个 u16 转换为一个 big endian formatted u8 vec
        let word_bytes: Vec<u8> = index.to_be_bytes().to_vec();
        
        // 将一个 big endian formatted u8 vec 转换为一个 big int
        let tmp_or = BigInt::from_bytes_be(Sign::Plus, &word_bytes);
        
        // 左移11位，腾出11位的空间来
        b = b * right_shift_11_bits_divider.clone();
        
        // 给最右边的11位空间进行赋值
        b = b | tmp_or;
    }
    
    // 从助记词+校验值组成的byte数组中计算出原始的随机熵
    let base: u32 = 2;
    let checksum_modulo = (base.pow(checksum_bit_size.try_into().unwrap())).to_bigint().unwrap();
    
    // 右移11位，来剔除掉校验位，获得原始的熵值
    let entropy: BigInt = &b / checksum_modulo;
//    println!("entropy:{:?}", entropy);
    
    // 校验位最多有8个比特，计算出完整的字节长度
    // 计算出被用来计算校验位的原始内容的字节长度
    let entropy_byte_size = (mnemonic_bit_size - checksum_bit_size) / 8;
    
    // 计算出包含计算出的校验位的内容的字节长度，校验位最多有8个比特，也就是一个字节
    let full_byte_size = entropy_byte_size + 1;
    
    let (_, entropy_bytes) = entropy.to_bytes_be();
    
    println!("entropy_bytes:{:?}", entropy_bytes);
    
    let (_, b_bytes) = b.to_bytes_be();
    
    let entropy_with_checksum_bytes = utils::bytes::bytes_pad(b_bytes, full_byte_size);
    
    // 检查校验位是否正确
    let mut entropy_bytes_after_checksum = add_checksum(entropy_bytes.clone());
    
//    println!("entropy_bytes after add_checksum:{:?}, full_byte_size:{:?}", entropy_bytes_after_checksum, full_byte_size);
    
    entropy_bytes_after_checksum = utils::bytes::bytes_pad(entropy_bytes_after_checksum, full_byte_size);
    
    println!("entropy_bytes after after_checksum and bytes_pad:{:?}", entropy_bytes_after_checksum);
    
    println!("entropy_with_checksum_bytes retrieved from mnemonic:{:?}", entropy_with_checksum_bytes);
    
    let cmp_result = utils::bytes::bytes_compare(entropy_with_checksum_bytes, entropy_bytes_after_checksum);
    
    if cmp_result != true {
        Err(
            CryptoError {
                kind: ErrorKind::MnemonicChecksumInvalid,
                message: ErrorKind::MnemonicChecksumInvalid.to_string()
            }
        )
    } else {
        Ok(entropy_bytes)
    }
}

#[allow(dead_code)]
/// 检查助记词字符串是否有效，如果有效，返回助记词的集合Vector
pub fn get_words_from_valid_mnemonic_sentence(mnemonic: String, language_type: wordlist::LanguageType) -> CryptoResult<Vec<String>> {
    // 将助记词字符串以空格符分割，返回包含助记词的list
    let words_result = get_words_from_mnemonic_sentence(mnemonic);
    
    // 判断是否从助记词字符串中成功的取出了符合数量要求的助记词
    let words = match words_result {
        Ok(wordlist) => wordlist,
        Err(error) => return Err(error),
    };
    
    // 根据指定的语言类型来选择助记词词库
    let mut wordlist = wordlist::Wordlist {
        language: language_type,
        word_list: Vec::new(),
        word_map: HashMap::new(),
    };
    
    wordlist::GetWord::get_wordlist(&mut wordlist)?;
    
    // 判断是否在对应语言的词库里
    let check_words_match_result = check_words_within_language_wordlist(words, wordlist.word_list);
    // 判断是否都在词库里
    match check_words_match_result {
        Ok(words) => return Ok(words),
        Err(error) => return Err(error),
    };
}

/// 内部函数
/// 计算 sha256(data)的前(len(data)/32)比特位的值作为校验码，
/// 并将其加到data后面，然后返回新的带有校验码的data
fn add_checksum(data: Vec<u8>) -> Vec<u8> {
    // 获取sha256处理后的第二个字节作为校验码
//    let data_hash = digest_bytes(&data);
//    let hash_byte = data_hash.as_bytes();

    let mut sha256 = Sha256::new();
    
    // SHA256 once
    sha256.input(&data);
    let mut hash_byte: [u8; 32] = [0; 32];
    sha256.result(&mut hash_byte);
    
    sha256.reset();

    let first_checksum_byte = hash_byte[1];
    
//    println!("first_checksum_byte:{:?}", first_checksum_byte);
    
    // CS = ENT / 32
    // len() 相当于/8，所以这里再除以4就行了
    // 计算出校验位的比特长度
    let checksum_bit_length = data.len() / 4;
    
    // The bytes are in big-endian byte order.
//    println!("data:{:?}", data);
//    let mut data_bigint = BigInt::from_bytes_be(&data);
    let mut data_bigint = BigInt::from_bytes_be(Sign::Plus, &data);
//    println!("data_bigint:{:?}", data_bigint);
    
    // 执行校验位长度N的循环，来生成长度N的校验位
    let mut index: usize = 0;
    
    while index < checksum_bit_length {
        // 乘以10等于比特位运算左移一位，将原始熵全部左移一位
        data_bigint = data_bigint * 2;
        
        // Set rightmost bit if leftmost checksum bit is set
        if first_checksum_byte & (1<<(7-index)) > 0 {
            // 与00000001进行或，相当于对最右边的那个比特位进行计算，算出校验位
            data_bigint = data_bigint | 1.to_bigint().unwrap();
        }        
        
        index += 1;
        
//        println!("data_bigint:{:?}", data_bigint);
    }
    
    let (_, data_bytes) = data_bigint.to_bytes_be();
    
    data_bytes
}

/// 内部函数
/// 取出助记词字符串中的所有助记词，并且同时检查助记词字符串包含的助记词数量是否有效
///  checksum length (CS)
///  entropy length (ENT)
///  mnemonic sentence (MS)
///
///  CS = ENT / 32
///  MS = (ENT + CS) / 11
///
///  |  ENT  | CS | ENT+CS |  MS  |
//  +-------+----+--------+------+
///  |  128  |  4 |   132  |  12  |
///  |  160  |  5 |   165  |  15  |
///  |  192  |  6 |   198  |  18  |
///  |  224  |  7 |   231  |  21  |
///  |  256  |  8 |   264  |  24  |
fn get_words_from_mnemonic_sentence(mnemonic: String) -> CryptoResult<Vec<String>> {
    // 将助记词字符串以空格符分割，返回包含助记词的vec
    let res: Vec<String> = mnemonic.split(" ").map(|x| x.to_string()).collect();
    
    //统计助记词的数量
    let num = res.len();
    
    // 助记词的数量只能是 12, 15, 18, 21, 24
    let valid_num_slice = vec![12, 15, 18, 21, 24];

    if !valid_num_slice.contains(&num) {
        Err(
            CryptoError {
                kind: ErrorKind::MnemonicNumInvalid,
                message: ErrorKind::MnemonicNumInvalid.to_string()
            }
        )
    } else {
        Ok(res)
    }
}

// 内部函数
/// 检查是否集合中所有的助记词都在合法的助记词字典中
fn check_words_within_language_wordlist(words: Vec<String>, wordlist: Vec<String>) -> CryptoResult<Vec<String>> {
//    let wordlist_iter = wordlist.iter();
    for word in &words {
//        if n.iter().any(|&i| i==word)
        if !wordlist.contains(word) {
            return Err(
                CryptoError {
                    kind: ErrorKind::MnemonicWordInvalid,
                    message: ErrorKind::MnemonicWordInvalid.to_string()
                }
            )
        }
    }
    
    Ok(words)
}
