#[allow(dead_code)]
mod encryption;

#[cfg(test)]
mod tests {
    use crate::encryption::RSAEncryptionManager;
    const PRIV_KEY: &str = r#"
-----BEGIN RSA PRIVATE KEY-----
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
-----END RSA PRIVATE KEY-----
"#;
    #[test]
    fn new_key_manager_works() {
        let _ = RSAEncryptionManager::init();
    }

    #[test]
    fn load_private_key_works() {
        let mut manager = RSAEncryptionManager::init();
        match manager.load_priv_key(PRIV_KEY) {
            crate::encryption::Outcome::Success => print!("this is successful"),
            _ => print!("it failed"),
        };
    }

    #[test]
    fn encryption_works() {
        let mut manager = RSAEncryptionManager::init();
        match manager.load_priv_key(PRIV_KEY) {
            crate::encryption::Outcome::Success => print!("this is successful"),
            _ => print!("it failed"),
        };
        let msg = "this is my message: get fucked";
        if let Some(_) = manager.encrypt(&msg.as_bytes()) {
            println!("success");
        } else {
            panic!("encryption failed")
        }
    }

    #[test]
    fn decryption_works() {
        let mut manager = RSAEncryptionManager::init();
        match manager.load_priv_key(PRIV_KEY) {
            crate::encryption::Outcome::Success => print!("this is successful"),
            _ => print!("it failed"),
        };
        let msg = "this is my message: get fucked";
        if let Some(encrypted_msg_bytes) = manager.encrypt(&msg.as_bytes()) {
            if let Some(decrypted_msg_bytes) = manager.decrypt(&encrypted_msg_bytes) {
                let decrypted_msg = String::from_utf8_lossy(&decrypted_msg_bytes);
                let raw_decrypted_msg = &*decrypted_msg;
                println!("message decrypté : {}", raw_decrypted_msg);
                assert_eq!(raw_decrypted_msg, msg);
            }
        }
    }
}
