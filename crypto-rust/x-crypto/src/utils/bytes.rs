use std::cmp::Ordering;

// 将两个字节数组的内容合并
pub fn bytes_combine(a_bytes: &[u8], b_bytes: &[u8]) -> Vec<u8> {
    let mut buffer = Vec::with_capacity(a_bytes.len() + b_bytes.len());
    
    buffer.extend_from_slice(a_bytes);
    buffer.extend_from_slice(b_bytes);
    
    buffer
}

/// 比较两个字节数组的内容是否完全一致
pub fn bytes_compare(a: Vec<u8>, b: Vec<u8>) -> bool {
    let cmp_result = a.cmp(&b);
    
    if cmp_result == Ordering::Equal {
        true
    } else {
        false
    }
}

/// 补齐vec的长度补齐到指定字节的长度，返回Vec<u8>，补齐方式是前方补零
pub fn bytes_pad(p_bytes: Vec<u8>, length: usize) -> Vec<u8> {
    let mut new_slice: Vec<u8> = Vec::new();

    let mut index = p_bytes.len();
    while index < length {
        let x: u8 = 0;
        new_slice.push(x);
        
        index += 1;
    }
    
//     println!("new_slice:{:?}", new_slice);
    
    let p_bytes_iter = p_bytes.iter();
    for value in p_bytes_iter {
        new_slice.push(*value);
        
//        println!("new_slice within round:{:?}", new_slice);
    }
    
//    println!("new_slice now:{:?}", new_slice);
    
    new_slice
}
