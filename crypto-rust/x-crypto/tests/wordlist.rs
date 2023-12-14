use std::collections::HashMap;

use x_crypto::wordlist::GetWord;
use x_crypto::wordlist::Wordlist;
use x_crypto::wordlist::LanguageType;


#[test]
fn get_eng_word_list() {
    // test eng wordlist
    let mut eng_wordlist = Wordlist {
        language: LanguageType::English,
        word_list: Vec::new(),
        word_map: HashMap::new(),
    };
    
    let res = eng_wordlist.get_wordlist();
    
    assert_eq!(res.ok(), Some(true));
    
    assert_eq!(eng_wordlist.word_list[0], "abandon");
}

#[test]
fn get_simplified_chinese_word_list() {
    // test simplified chinese wordlist
    let mut simplified_chinese_wordlist = Wordlist {
        language: LanguageType::SimplifiedChinese,
        word_list: Vec::new(),
        word_map: HashMap::new(),
    };
    
    let res = simplified_chinese_wordlist.get_wordlist();
    
    assert_eq!(res.ok(), Some(true));
    
    assert_eq!(simplified_chinese_wordlist.word_list[0], "泊");
}

#[test]
fn get_eng_word_map() {
    // test eng wordlist
    let mut eng_wordlist = Wordlist {
        language: LanguageType::English,
        word_list: Vec::new(),
        word_map: HashMap::new(),
    };
    
    let res = eng_wordlist.get_wordlist();
    assert_eq!(res.ok(), Some(true));
    
    let res = eng_wordlist.get_reversed_wordmap();
    assert_eq!(res.ok(), Some(true));
    assert_eq!(eng_wordlist.word_map["ability"], 1);
}

#[test]
fn get_simplified_chinese_word_map() {
    // test simplified chinese wordlist
    let mut simplified_chinese_wordlist = Wordlist {
        language: LanguageType::SimplifiedChinese,
        word_list: Vec::new(),
        word_map: HashMap::new(),
    };
    
    let res = simplified_chinese_wordlist.get_wordlist();
    assert_eq!(res.ok(), Some(true));
    
    let res = simplified_chinese_wordlist.get_reversed_wordmap();
    assert_eq!(res.ok(), Some(true));
    assert_eq!(simplified_chinese_wordlist.word_map["帅"], 3);
}