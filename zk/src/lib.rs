use bellman::groth16::{prepare_verifying_key, verify_proof, Proof, VerifyingKey};
use bls12_381::Bls12;
use libc;

#[no_mangle]
pub extern "C" fn verify(
    proof: *mut libc::c_uchar,
    proof_len: libc::size_t,
    key: *mut libc::c_uchar,
    key_len: libc::size_t,
) -> libc::c_int {
    let bproof = unsafe { std::slice::from_raw_parts(proof, proof_len) };
    let bkey = unsafe { std::slice::from_raw_parts(key, key_len) };
    let rproof = Proof::<Bls12>::read(bproof);
    if let Ok(tproof) = rproof {
        let rvk = VerifyingKey::read(bkey);
        if let Ok(tvk) = rvk {
            let tpvk = prepare_verifying_key(&tvk);
            let result = verify_proof(&tpvk, &tproof, &[]);
            return match result {
                Err(_) => 0,
                Ok(_) => 1,
            };
        }
    }
    return 0;
}
