use crate::my_error::CryptoResult;
use crate::mnemonic;

use crypto::pbkdf2;
use crypto::sha2::Sha512;
use crypto::hmac::Hmac;

//use rand::Rng;

/// 产生指定长度的随机熵
pub fn generate_entropy(bit_size: usize) -> CryptoResult<Vec<u8>> {
    // 校验一遍熵的长度是否符合规范
    mnemonic::validate_raw_entropy_bit_size(bit_size)?;
    
    let entropy_size = bit_size / 8;
    
//    let entropy = rand::thread_rng().gen::<[u8; entropy_size]>();
    
    let entropy = (0..entropy_size).map(|_| { rand::random::<u8>() }).collect();

    Ok(entropy)
}

/// 根据助记词和用户指定的密码，产生长度为40*8的伪随机数，伪随机数算法使用pbkdf2
pub fn generate_seed_from_mnemonic(mnemonic: String, password: String) -> [u8; 40] {
    // 其实把助记词当password传入Hmac
    let mut mac = Hmac::new(Sha512::new(), mnemonic.as_bytes());
    
    let salt = String::from("mnemonic") + &password;
    let salt_bytes = salt.as_bytes();

    // initialize an array of 40*8 length
    let mut output: [u8; 40] = [0; 40];
    
    let round: u32 = 2048;
    
    pbkdf2::pbkdf2(&mut mac, salt_bytes, round, &mut output);
    
//    let output = output.to_vec();
    
    output
}