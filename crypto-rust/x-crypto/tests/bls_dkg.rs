use x_crypto::bls::threshold::bls_dkg;

#[test]
fn create_new_bls_account() {
    // Generate a BLS key pair
    let bls_account1 = bls_dkg::create_new_bls_account();

    println!("bls_account1: {:?}", bls_account1);
    
    let bls_account2 = bls_dkg::create_new_bls_account();

    println!("bls_account2: {:?}", bls_account2);
}