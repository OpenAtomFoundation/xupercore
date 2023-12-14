use x_crypto::account;
//use crypto::my_error;
use x_crypto::wordlist::LanguageType;
use x_crypto::crypto_config;


#[test]
fn get_account() {
    // 尝试创建新账户
    let res = account::create_new_account_with_mnemonic(LanguageType::English, account::MnemonicStrength::StrengthEasy, crypto_config::CryptoType::Gm);
        
//    println!("create_new_account_with_mnemonic -- account: {:?}", res);
    
//    let act = account::Account {
//        entropy:  Vec::new(),
//        mnemonic: mnemonic_sentence,
//        address: String::from(""),
//        json_private_key: String::from(""),
//        json_public_key: String::from(""),
//    };

    let error_msg = match res {
        Ok(_) => String::from("everything is fine."),
        Err(error) => error.to_string(),
    };
    
    assert_eq!(error_msg, "everything is fine.");
}

#[test]
fn retrieve_account() {
//    let mnemonic_sentence = String::from("呈 仓 冯 滚 刚 伙 此 丈 锅 语 揭 弃 精 塘 界 戴 玩 爬 奶 滩 哀 极 样 费");
//    
//    let account = account::retrieve_account_by_mnemonic(mnemonic_sentence, LanguageType::SimplifiedChinese).unwrap();

//    let mnemonic_sentence = String::from("abandon ability able about above absent absorb abstract absurd abuse access accident");
    
//    let mnemonic_sentence = String::from("evil reduce stereo video casual wonder kitchen exit jealous nuclear rural cactus");

    let mnemonic_sentence = String::from("pilot soft canal assault once puppy pole cross defy extend civil camp");
    
    let account = account::retrieve_account_by_mnemonic(mnemonic_sentence, LanguageType::English).unwrap();
    
    println!("account: {:?}", account);
    
    let address_expect = String::from("ZXG4hvkFjB5yJ71wNo6YT5uR93fuHSuzo");
    
    assert_eq!(address_expect, account.address);
}

#[test]
fn retrieve_secret_key() {
    // 创建新账户
    let account = account::create_new_account_with_mnemonic(LanguageType::English, account::MnemonicStrength::StrengthEasy, crypto_config::CryptoType::Gm).unwrap();
    
    // 获取使用PEM-encoded SEC1格式编码的私钥字符串
    let secret_key_str = account.private_key;
    
    // 将使用PEM-encoded SEC1格式编码的私钥字符串转化为ECC私钥
    let secret_key_parse_result = account::get_ecdsa_private_key_from_pem_encoded_sec1_str(secret_key_str);
    
    println!("secret_key_parse_result: {:?}", secret_key_parse_result);
    
    let error_msg = match secret_key_parse_result {
        Ok(_) => String::from("everything is fine."),
        Err(error) => error.to_string(),
    };
    
    assert_eq!(error_msg, "everything is fine.");
}

#[test]
fn retrieve_public_key() {
    // 创建新账户
    let account = account::create_new_account_with_mnemonic(LanguageType::English, account::MnemonicStrength::StrengthEasy, crypto_config::CryptoType::Gm).unwrap();
    
    // 获取使用PEM-encoded SEC1格式编码的公钥字符串
    let public_key_str = account.public_key;
    
    // 将使用PEM-encoded SEC1格式编码的公钥字符串转化为ECC公钥
    let public_key_parse_result = account::get_ecdsa_public_key_from_pem_encoded_sec1_str(public_key_str);
    
    println!("public_key_parse_result: {:?}", public_key_parse_result);
    
    let error_msg = match public_key_parse_result {
        Ok(_) => String::from("everything is fine."),
        Err(error) => error.to_string(),
    };
    
    assert_eq!(error_msg, "everything is fine.");
}


#[test]
fn ecdsa() {
    // 创建新账户
    let res = account::create_new_account_with_mnemonic(LanguageType::English, account::MnemonicStrength::StrengthEasy, crypto_config::CryptoType::Gm);

    let account = res.unwrap();
    
    // 获取使用PEM-encoded SEC1格式编码的私钥字符串
    let secret_key_str = account.private_key;
    let data = String::from("this is a test string");
    let sig = account::ecdsa_sign(secret_key_str, data).unwrap();
    let sig_copy = sig.clone();
    
    println!("ecdsa sig: {:?}", sig);
    
//    let sig = sig_result.unwrap();
    
    // 获取使用PEM-encoded SEC1格式编码的公钥字符串
    let public_key_str = account.public_key.clone();
    let data_verify = String::from("this is a test string");
    let verify_result = account::ecdsa_verify(public_key_str, data_verify, sig);
    
    println!("ecdsa verify_result: {:?}", verify_result);
    
    let res = match verify_result {
        Ok(result) => result,
        Err(_) => false,
    };
    assert_eq!(res, true);
    
    let public_key_str = account.public_key;
    let data_verify_false = String::from("this is another test string");
    let verify_result = account::ecdsa_verify(public_key_str, data_verify_false, sig_copy);
    println!("ecdsa verify_result for wrong data: {:?}", verify_result);
    let res = match verify_result {
        Ok(result) => result,
        Err(_) => false,
    };
    assert_eq!(res, false);

}