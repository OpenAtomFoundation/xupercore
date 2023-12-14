use crate::my_error::{ErrorKind,CryptoError,CryptoResult};
use crate::ecies::params::ECIESParams;

use aes_gcm::aead::{generic_array::GenericArray, AeadInPlace};
use aes_gcm::{aes::Aes128, AesGcm, KeyInit};
use rand::{thread_rng, Rng};
use typenum::consts::U16;

/// AES-128-GCM with 16 bytes Nonce/IV
pub type Aes128Gcm = AesGcm<Aes128, U16>;

/// AES-128-GCM encryption wrapper
pub fn aes_encrypt(key: &[u8], msg: &[u8], params: ECIESParams) -> CryptoResult<Vec<u8>> {
    let key = GenericArray::from_slice(key);
    let aead = Aes128Gcm::new(key);

//    let mut iv = [0u8; params.block_size];
//    let mut iv = Vec::with_capacity(params.block_size);
    let mut iv = vec![0u8; params.block_size];
    thread_rng().fill(&mut iv[..]);

    let nonce = GenericArray::from_slice(&iv);

    let mut out = Vec::with_capacity(msg.len());
//    let mut out = vec![0u8; msg.len()];
    out.extend(msg);

    // Additional Authenticated Data 这里不需要，创建一个空数组
    let aad_bytes: [u8; 0] = [];

    if let Ok(tag) = aead.encrypt_in_place_detached(nonce, &aad_bytes, &mut out) {
        let mut output = Vec::with_capacity(params.block_size + params.key_len + msg.len());
        output.extend(&iv);
        output.extend(tag);
        output.extend(out);
        Ok(output)
    } else {
        return Err(
            CryptoError {
                kind: ErrorKind::EciesEncryptError,
                message: ErrorKind::EciesEncryptError.to_string()
            }
        );
    }
}

/// AES-128-GCM decryption wrapper
pub fn aes_decrypt(key: &[u8], encrypted_msg: &[u8], params: ECIESParams) -> CryptoResult<Vec<u8>> {
    if encrypted_msg.len() < params.block_size + params.key_len {
        return Err(
            CryptoError {
                kind: ErrorKind::EciesDecryptError,
                message: ErrorKind::EciesDecryptError.to_string()
            }
        );
    }

    let key = GenericArray::from_slice(key);
    let aead = Aes128Gcm::new(key);

    let iv = GenericArray::from_slice(&encrypted_msg[..params.block_size]);
    let tag = GenericArray::from_slice(&encrypted_msg[params.block_size..params.block_size + params.key_len]);

    let mut output = Vec::with_capacity(encrypted_msg.len() - params.block_size + params.key_len);
    output.extend(&encrypted_msg[params.block_size + params.key_len..]);

    // Additional Authenticated Data 这里不需要，创建一个空数组
    let aad_bytes: [u8; 0] = [];
    
    if let Ok(_) = aead.decrypt_in_place_detached(iv, &aad_bytes, &mut output, tag) {
        Ok(output)
    } else {
        return Err(
            CryptoError {
                kind: ErrorKind::EciesDecryptError,
                message: ErrorKind::EciesDecryptError.to_string()
            }
        );
    }
}