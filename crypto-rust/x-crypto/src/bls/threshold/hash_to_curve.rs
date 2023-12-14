use bls12_381::{
    G1Affine,
    Scalar
};

use crypto::{
    digest::Digest,
    sha2::Sha512    
};

// Hash a message to G1 curve
pub fn hash_to_g1_curve(message: &[u8]) -> G1Affine {
    let mut sha512 = Sha512::new();
    sha512.input(message);
    
    let mut hash_byte_sha512: [u8; 64] = [0; 64];
    sha512.result(&mut hash_byte_sha512);
    
    // Converts a 512-bit little endian integer into a Scalar by reducing by the modulus.
    let scalar_factor = Scalar::from_bytes_wide(&hash_byte_sha512);
    
    G1Affine::from(G1Affine::generator() * scalar_factor)
}