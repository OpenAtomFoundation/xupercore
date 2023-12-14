//use serde_json;
use serde::{Deserialize, Serialize};

#[derive(Serialize, Deserialize)]
// 通过这个数据结构来生成私钥的json
pub struct ECDSAPrivateKey {
    pub curvname: String,
    pub x: String, // 第三方库的bigint，在这里，不支持序列化与反序列化
    pub y: String,
    pub d: String,
}