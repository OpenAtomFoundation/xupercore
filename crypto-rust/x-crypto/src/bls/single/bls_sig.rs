use ff::Field;
use bls12_381::{G1Affine, G2Affine, Scalar};

use crypto::sha2::Sha512;
use crypto::digest::Digest;

use rand::rngs::OsRng;

// Generate a BLS key pair
pub fn generate_key_pair() -> (Scalar, G2Affine) {
    let mut rng = OsRng {};
    
    let private_key  = Scalar::random(&mut rng);
    let public_key = G2Affine::from(G2Affine::generator() * private_key);
    (private_key, public_key)
}

// Sign a message using BLS
pub fn sign(message: &[u8], private_key: Scalar) -> G1Affine {
    let mut sha512 = Sha512::new();
    sha512.input(message);
    
    let mut hash_byte_sha512: [u8; 64] = [0; 64];
    sha512.result(&mut hash_byte_sha512);
    
    // Converts a 512-bit little endian integer into a Scalar by reducing by the modulus.
    let scalar_factor = Scalar::from_bytes_wide(&hash_byte_sha512);
    
    G1Affine::from(G1Affine::generator() * (private_key * scalar_factor))
}

// Verify a BLS signature
pub fn verify(message: &[u8], signature: G1Affine, public_key: G2Affine) -> bool {
    let mut sha512 = Sha512::new();
    sha512.input(message);
    
    let mut hash_byte_sha512: [u8; 64] = [0; 64];
    sha512.result(&mut hash_byte_sha512);
    
    let scalar_factor = Scalar::from_bytes_wide(&hash_byte_sha512);
    
    let g2_generator = G2Affine::generator();
    let left = bls12_381::pairing(&signature, &g2_generator);
    
    let g2_verify_point = G2Affine::from(public_key * scalar_factor);
    let right = bls12_381::pairing(&G1Affine::generator(), &g2_verify_point);
    
    left == right
}