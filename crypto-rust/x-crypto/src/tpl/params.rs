#[derive(Clone, Debug)]
pub struct TplParams {
    // third party label
    pub tpl: String,
    // secret key used for generating signatures
    pub sk: String,
}

/// 获取连接matrix dendrite服务器的配置参数
pub fn get_tpl_params() -> TplParams {
    let tpl_params: TplParams = TplParams{
            tpl: String::from("xsocial"),
            sk: String::from("F1G9A82BAF4F9O10BDD543A7D1FRB786"),
        };
    
    tpl_params
}