use std::collections::HashMap;
use crate::language_dict::english_dict;
use crate::language_dict::simplified_chinese_dict;
use crate::my_error::{ErrorKind,CryptoError,CryptoResult};

pub trait GetWord {
    fn get_wordlist(&mut self) -> CryptoResult<bool>; 
//    fn get_wordlist(&self) -> Result<Vec<&str>, LanguageError>;
    fn get_reversed_wordmap(&mut self) -> CryptoResult<bool>; 
}

#[derive(Clone, Debug, Copy)]
pub enum LanguageType {
    SimplifiedChinese,
    English,
}

//pub struct Wordlist<'a> {
////    pub word_dict: &str,
//    pub language: LanguageType,
//    pub word_list: Vec<&'a str>,
//}

#[derive(Clone, Debug)]
pub struct Wordlist {
//    pub word_dict: &str,
    pub language: LanguageType,
    pub word_list: Vec<String>,
    pub word_map: HashMap<String, u16>,
}

impl GetWord for Wordlist {
        fn get_wordlist(&mut self) -> CryptoResult<bool> {
            let language_type = &self.language;
            
//            let mut v: Vec<&str> = Vec::new();

            match language_type {
                LanguageType::English => {
//                    self.word_list = english_dict::get_english_wordlist_dict().split("\n").collect::<Vec<&str>>();
                    self.word_list = english_dict::get_english_wordlist_dict().split("\n").map(|x| x.to_string()).collect();
                    Ok(true)
                }
                LanguageType::SimplifiedChinese => {
//                    self.word_list = simplified_chinese_dict::get_simplified_chinese_wordlist_dict().split("\n").collect::<Vec<&str>>();
                    self.word_list = simplified_chinese_dict::get_simplified_chinese_wordlist_dict().split("\n").map(|x| x.to_string()).collect();
                    Ok(true)
                }
                // 为之后扩展做准备
                #[allow(unreachable_patterns)]
                _ => {
                    Err(
                        CryptoError {
                            kind: ErrorKind::LanguageNotSupportedYet,
                            message: ErrorKind::LanguageNotSupportedYet.to_string()
                        }
                    )
                }
            }
        }
    
    fn get_reversed_wordmap(&mut self) -> CryptoResult<bool> {
        if self.word_list.len() == 0 {
            Err(
                CryptoError {
                    kind: ErrorKind::WordlistNotInitiatedYet,
                    message: ErrorKind::WordlistNotInitiatedYet.to_string()
                }
            )
        } else {
            let mut word_map = HashMap::<String, u16>::new();
            for (index, word) in self.word_list.iter().enumerate() {
                // todo: 注意这里的实现不优雅，理论上可能会panic。但是实际上不可能panic...
                word_map.insert(word.clone(), index.try_into().unwrap());
            }
        
            self.word_map = word_map;
            
            Ok(true)
        }
    }
    
}