use std::collections::HashMap;

use x_crypto::mnemonic;
use x_crypto::my_error;

use x_crypto::wordlist::GetWord;
use x_crypto::wordlist::Wordlist;
use x_crypto::wordlist::LanguageType;



#[test]
fn get_words_from_mnemonic() {
    // test eng wordlist
    let mut eng_wordlist = Wordlist {
        language: LanguageType::English,
        word_list: Vec::new(),
        word_map: HashMap::new(),
    };
    
    let res = eng_wordlist.get_wordlist();
    assert_eq!(res.ok(), Some(true));
    
    // 12 vaild words
    let mnemonic = String::from("abandon ability able about above absent absorb abstract absurd abuse access accident");

    let res = mnemonic::get_words_from_valid_mnemonic_sentence(mnemonic, LanguageType::English);
    
    let words: Vec<String> = vec!["abandon".to_string(),"ability".to_string(),"able".to_string(),
        "about".to_string(),"above".to_string(),"absent".to_string(),"absorb".to_string(),"abstract".to_string()
        ,"absurd".to_string(),"abuse".to_string(),"access".to_string(),"accident".to_string()];
        
    println!("{:?}", res);
        
    assert_eq!(res.ok(), Some(words));
    
    // 11 vaild words, 1 invalid word
    let mnemonic = String::from("abandon ability able about above absent absorb abstract absurd abuse access accidents");

    let res = mnemonic::get_words_from_valid_mnemonic_sentence(mnemonic, LanguageType::English);
        
    println!("{:?}", res);
    
    let error_msg = match res {
        Ok(_) => String::from("everything is fine."),
        Err(error) => error.to_string(),
    };
        
    assert_eq!(error_msg, my_error::ErrorKind::MnemonicWordInvalid.to_string());
}

#[test]
fn get_entropy_from_mnemonic() {
    let mnemonic = String::from("evil reduce stereo video casual wonder kitchen exit jealous nuclear rural cactus");
    
    let language_type = LanguageType::English;
    
    let res = mnemonic::get_entropy_from_mnemonic_sentence(mnemonic, language_type);
    
    println!("{:?}", res);
    
    assert_eq!(res.ok(), Some(vec![77, 246, 131, 86, 121, 226, 59, 250, 30, 194, 127, 119, 146, 234, 246, 16]));
}

#[test]
fn get_mnemonic_from_entropy() {
    let entropy = vec![77, 246, 131, 86, 121, 226, 59, 250, 30, 194, 127, 119, 146, 234, 246, 16];
    
    let mnemonic = String::from("evil reduce stereo video casual wonder kitchen exit jealous nuclear rural cactus");
    
    let language_type = LanguageType::English;
    
    let res = mnemonic::generate_mnemonic_sentence_from_entropy(entropy, language_type);
    
    println!("{:?}", res);
    
    assert_eq!(res.ok(), Some(mnemonic));
}
