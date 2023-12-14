use x_crypto::bls::threshold::{
    bls_dkg,
    bls_dsg,
};

#[test]
fn bls_threshold_sign() {
    // --- DKG start ---
    
    // Generate a BLS key pair
    let bls_account1 = bls_dkg::create_new_bls_account();
    println!("bls_account1: {:?}", bls_account1);
    
    let bls_account2 = bls_dkg::create_new_bls_account();
    println!("bls_account2: {:?}", bls_account2);
    
    let bls_account3 = bls_dkg::create_new_bls_account();
    println!("bls_account3: {:?}", bls_account3);
    
    // 计算公钥之和
    let mut bls_public_keys: Vec<bls_dkg::BlsPublicKey> = Vec::new();
    bls_public_keys.push(bls_account1.public_key);
    bls_public_keys.push(bls_account2.public_key);
    bls_public_keys.push(bls_account3.public_key);
    
    let bls_public_key_sum = bls_dkg::sum_bls_public_key(&bls_public_keys).unwrap();
    
    let bls_k1 = bls_dkg::get_k(bls_account1.public_key, bls_public_key_sum);
    let bls_k2 = bls_dkg::get_k(bls_account2.public_key, bls_public_key_sum);
    let bls_k3 = bls_dkg::get_k(bls_account3.public_key, bls_public_key_sum);

    let bls_public_key_part1 = bls_dkg::get_public_key_part(bls_account1.public_key, bls_k1);
    let bls_public_key_part2 = bls_dkg::get_public_key_part(bls_account2.public_key, bls_k2);
    let bls_public_key_part3 = bls_dkg::get_public_key_part(bls_account3.public_key, bls_k3);
    
    // 计算公钥碎片之和
    let mut bls_public_key_parts: Vec<bls_dkg::BlsPublicKey> = Vec::new();
    bls_public_key_parts.push(bls_public_key_part1);
    bls_public_key_parts.push(bls_public_key_part2);
    bls_public_key_parts.push(bls_public_key_part3);
    
    let bls_public_key_share = bls_dkg::sum_bls_public_key(&bls_public_key_parts).unwrap();
    
    let bls_node1_m1 = bls_dkg::get_m(bls_k1, bls_account1.private_key.x, bls_account1.index.to_owned(), bls_public_key_share);
    let bls_node1_m2 = bls_dkg::get_m(bls_k1, bls_account1.private_key.x, bls_account2.index.to_owned(), bls_public_key_share);
    let bls_node1_m3 = bls_dkg::get_m(bls_k1, bls_account1.private_key.x, bls_account3.index.to_owned(), bls_public_key_share);
    
    let bls_node2_m1 = bls_dkg::get_m(bls_k2, bls_account2.private_key.x, bls_account1.index.to_owned(), bls_public_key_share);
    let bls_node2_m2 = bls_dkg::get_m(bls_k2, bls_account2.private_key.x, bls_account2.index.to_owned(), bls_public_key_share);
    let bls_node2_m3 = bls_dkg::get_m(bls_k2, bls_account2.private_key.x, bls_account3.index.to_owned(), bls_public_key_share);
    
    let bls_node3_m1 = bls_dkg::get_m(bls_k3, bls_account3.private_key.x, bls_account1.index.to_owned(), bls_public_key_share);
    let bls_node3_m2 = bls_dkg::get_m(bls_k3, bls_account3.private_key.x, bls_account2.index.to_owned(), bls_public_key_share);
    let bls_node3_m3 = bls_dkg::get_m(bls_k3, bls_account3.private_key.x, bls_account3.index.to_owned(), bls_public_key_share);
    
    let mut ms1: Vec<bls_dkg::BlsM> = Vec::new();
    ms1.push(bls_node1_m1);
    ms1.push(bls_node2_m1);
    ms1.push(bls_node3_m1);
    
    let mk1 = bls_dkg::get_mk(&ms1).unwrap();
    println!("mk1 is: {:?}", mk1.p);
    
    let mut ms2: Vec<bls_dkg::BlsM> = Vec::new();
    ms2.push(bls_node1_m2);
    ms2.push(bls_node2_m2);
    ms2.push(bls_node3_m2);
    
    let mk2 = bls_dkg::get_mk(&ms2).unwrap();
    println!("mk2 is: {:?}", mk2.p);
    
    let mut ms3: Vec<bls_dkg::BlsM> = Vec::new();
    ms3.push(bls_node1_m3);
    ms3.push(bls_node2_m3);
    ms3.push(bls_node3_m3);
    
    let mk3 = bls_dkg::get_mk(&ms3).unwrap();
    println!("mk3 is: {:?}", mk3.p);
    
    let is_mk1_right = bls_dkg::verify_mk(bls_public_key_share, bls_account1.index.to_owned(), mk1);
    println!("mk1 is right or not: {}", is_mk1_right);
    
    let is_mk1_right = bls_dkg::verify_mk_by_multi_miller_loop(bls_public_key_share, bls_account1.index.to_owned(), mk1);
    println!("mk1 is right or not by [verify_mk_by_multi_miller_loop]: {}", is_mk1_right);
    
    let is_mk2_right = bls_dkg::verify_mk_by_multi_miller_loop(bls_public_key_share, bls_account2.index.to_owned(), mk2);
    println!("mk2 is right or not by [verify_mk_by_multi_miller_loop]: {}", is_mk2_right);
    
    let is_mk3_right = bls_dkg::verify_mk_by_multi_miller_loop(bls_public_key_share, bls_account3.index.to_owned(), mk3);
    println!("mk3 is right or not by [verify_mk_by_multi_miller_loop]: {}", is_mk3_right);

    // --- DKG end ---
    
    // --- DSG start ---
    
    let msg = "the msg for bls threshold schema".as_bytes();

    let partner_public_1 = bls_dkg::PartnerPublic {
        index: bls_account1.index,
        public_key: bls_account1.public_key,
    };
    let partner_private_1 = bls_dkg::PartnerPrivate {
        public_info: partner_public_1,
        threshold_public_key: bls_public_key_share,
        x: bls_account1.private_key.x,
        mki: mk1.p,
    };
    let bls_signature_part_1 = bls_dsg::sign(partner_private_1, msg);
    
    let partner_public_3 = bls_dkg::PartnerPublic {
        index: bls_account3.index,
        public_key: bls_account3.public_key,
    };
    let partner_private_3 = bls_dkg::PartnerPrivate {
        public_info: partner_public_3,
        threshold_public_key: bls_public_key_share,
        x: bls_account3.private_key.x,
        mki: mk3.p,
    };
    let bls_signature_part_3 = bls_dsg::sign(partner_private_3, msg);
    
    let mut bls_signature_parts: Vec<bls_dsg::BlsSignaturePart> = Vec::new();
    bls_signature_parts.push(bls_signature_part_1);
    bls_signature_parts.push(bls_signature_part_3);
    
    let bls_threshold_signature = bls_dsg::combine_sign(&bls_signature_parts);
    
    let is_signature_match = bls_dsg::verify_sign(bls_public_key_share, bls_threshold_signature, msg);
    println!("Verifying BLS threshold signature using bls_public_key_share, is_signature_match is {}", is_signature_match);

    assert_eq!(is_signature_match, true);
    
    // --- DSG end ---
}