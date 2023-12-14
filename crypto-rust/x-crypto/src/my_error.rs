use std::fmt;
use std::io;
use std::num;
use std::net;
use std::result::Result;

pub type CryptoResult<T> = Result<T, CryptoError>;

// Define our error types. These may be customized for our error handling cases.
// Now we will be able to write our own errors, defer to an underlying error
// implementation, or do something in between.

// Rust standard library provides not only reusable traits,
// and also it facilitates to magically generate implementations
// for few traits via #[derive] attribute. Rust support derive std::fmt::Debug,
// to provide a default format for debug messages.
// So we can skip std::fmt::Debug implementation for custom error types,
// and use #[derive(Debug)] before struct declaration.

// Debug means:
// How should display the Err while debugging/ programmer-facing output.
#[derive(Debug)]
pub struct CryptoError {
//    kind: String, 
    pub kind: ErrorKind,
    pub message: String,
}

#[derive(Debug)]
pub enum ErrorKind {
    /// The language is not supported yet.
    LanguageNotSupportedYet,
    /// The language is not supported yet.
    WordlistNotInitiatedYet,
    /// IO error.
    IoError,
    /// Parse int error.
    ParseIntError,
    /// Addr parse error.
    AddrParseError,
    /// Mnemonic number is invalid. Must within [12, 15, 18, 21, 24]
    MnemonicNumInvalid,
    /// Mnemonic is invalid. Contains illegal word.
    MnemonicWordInvalid,
    /// Mnemonic checksum is invalid. Contains illegal word.
    MnemonicChecksumInvalid,
    /// Entropy length is invalid. Must within [128, 160, 192, 224, 256]
    ErrInvalidEntropyLength,
    /// Raw Entropy length is invalid. Must within [120, 152, 184, 216, 248]. After +8 be multiples of 32
    ErrInvalidRawEntropyLength,
    /// Invalid String Format, failed when trying to parse to the target object. 
    ErrInvalidStringFormat,
    /// Invalid ECDSA Signature, failed when trying to verify the signature. 
    ErrInvalidEcdsaSig,
    /// Elliptic curve is not supported yet. 
    EllipticCurveNotSupportedYet,
    /// ecies shared key params are too big.
    EciesSharedKeyTooBig,
    /// Invalid SEC 1: Elliptic Curve Cryptography (Version 2.0) section 2.3.3 (page 10) Format,
    /// failed when trying to parse to the ECC key. 
    InvalidSecKeyFormat,
    /// failed when trying to do the HKDF process. 
    EciesHkdfInvalidKeyLength,
    /// failed when trying to do the ecies aes encrypt process. 
    EciesEncryptError,
    /// failed when trying to do the ecies aes decrypt process. 
    EciesDecryptError,
    /// ECC public key retrieve error. 
    EciesPublickeyRetrieveError,
    /// empty array error
    ErrEmptyArray,
    
//    /// Others.
//    Unknown,
}

// Generation of an error is completely separate from how it is displayed.
// There's no need to be concerned about cluttering complex logic with the display style.
//
// Note that we don't store any extra info about the errors. This means we can't state
// which string failed to parse without modifying our types to carry that information.

// Display means:
// How should the end user see this error as a message/ user-facing output.
impl fmt::Display for CryptoError {
    fn fmt(&self, f: &mut fmt::Formatter) -> fmt::Result {
        let err_msg = match self.kind {
            ErrorKind::LanguageNotSupportedYet => "The language is not supported yet!",
            ErrorKind::WordlistNotInitiatedYet => "WordlistNotInitiatedYet!",
            ErrorKind::IoError => "IoError!",
            ErrorKind::ParseIntError => "ParseIntError!",
            ErrorKind::AddrParseError => "AddrParseError!",
            ErrorKind::MnemonicNumInvalid => "MnemonicNumInvalid!",
            ErrorKind::MnemonicWordInvalid => "MnemonicWordInvalid!",
            ErrorKind::MnemonicChecksumInvalid => "MnemonicChecksumInvalid!",
            ErrorKind::ErrInvalidEntropyLength => "ErrInvalidEntropyLength!",
            ErrorKind::ErrInvalidRawEntropyLength => "ErrInvalidRawEntropyLength!",
            ErrorKind::ErrInvalidStringFormat => "ErrInvalidStringFormat!",
            ErrorKind::ErrInvalidEcdsaSig => "ErrInvalidEcdsaSig!",
            ErrorKind::EllipticCurveNotSupportedYet => "EllipticCurveNotSupportedYet!",
            ErrorKind::EciesSharedKeyTooBig => "ecies: shared key params are too big!",
            ErrorKind::InvalidSecKeyFormat => "ecies: InvalidSecKeyFormat!",
            ErrorKind::EciesHkdfInvalidKeyLength => "ecies: EciesHkdfInvalidKeyLength!",
            ErrorKind::EciesEncryptError => "ecies: EciesEncryptError!",
            ErrorKind::EciesDecryptError => "ecies: EciesDecryptError!",
            ErrorKind::EciesPublickeyRetrieveError => "ecies: EciesPublickeyRetrieveError!",
            ErrorKind::ErrEmptyArray => "ErrEmptyArray!",
//            _ => "Unkonwn Error!",
        };

//        let err_msg = &self.message;

        write!(f, "{}", err_msg)
    }
}

// Display means:
// How should the end user see this error as a message/ user-facing output.
impl fmt::Display for ErrorKind {
    fn fmt(&self, f: &mut fmt::Formatter) -> fmt::Result {
        let err_msg = match self {
            ErrorKind::LanguageNotSupportedYet => "The language is not supported yet!",
            ErrorKind::WordlistNotInitiatedYet => "WordlistNotInitiatedYet!",
            ErrorKind::IoError => "IoError!",
            ErrorKind::ParseIntError => "ParseIntError!",
            ErrorKind::AddrParseError => "AddrParseError!",
            ErrorKind::MnemonicNumInvalid => "MnemonicNumInvalid!",
            ErrorKind::MnemonicWordInvalid => "MnemonicWordInvalid!",
            ErrorKind::MnemonicChecksumInvalid => "MnemonicChecksumInvalid!",
            ErrorKind::ErrInvalidEntropyLength => "ErrInvalidEntropyLength!",
            ErrorKind::ErrInvalidRawEntropyLength => "ErrInvalidRawEntropyLength!",
            ErrorKind::ErrInvalidStringFormat => "ErrInvalidStringFormat!",
            ErrorKind::ErrInvalidEcdsaSig => "ErrInvalidEcdsaSig!",
            ErrorKind::EllipticCurveNotSupportedYet => "EllipticCurveNotSupportedYet!",
            ErrorKind::EciesSharedKeyTooBig => "ecies: shared key params are too big!",
            ErrorKind::InvalidSecKeyFormat => "ecies: InvalidSecKeyFormat!",
            ErrorKind::EciesHkdfInvalidKeyLength => "ecies: EciesHkdfInvalidKeyLength!",
            ErrorKind::EciesEncryptError => "ecies: EciesEncryptError!",
            ErrorKind::EciesDecryptError => "ecies: EciesDecryptError!",
            ErrorKind::EciesPublickeyRetrieveError => "ecies: EciesPublickeyRetrieveError!",
            ErrorKind::ErrEmptyArray => "ErrEmptyArray!",
//            _ => "Unkonwn Error!",
        };

        write!(f, "{}", err_msg)
    }
}

// Have to deal with different modules, different std and third party crates at the same time.
// Each crate uses their own error types. However, if using self defined error types,
// those errors should be converted into our error types. 
// For these conversions, we can use the standardized trait std::convert::From.

// Implement std::convert::From for CryptoError; from io::Error
impl From<io::Error> for CryptoError {
    fn from(error: io::Error) -> Self {
        CryptoError {
//            kind: String::from("io"),
            kind: ErrorKind::IoError,
            message: error.to_string(),
        }
    }
}

// Implement std::convert::From for CryptoError; from num::ParseIntError
impl From<num::ParseIntError> for CryptoError {
    fn from(error: num::ParseIntError) -> Self {
        CryptoError {
            kind: ErrorKind::ParseIntError,
            message: error.to_string(),
        }
    }
}

// Implement std::convert::From for CryptoError; from net::AddrParseError
impl From<net::AddrParseError> for CryptoError {
    fn from(error: net::AddrParseError) -> Self {
        CryptoError {
            kind: ErrorKind::AddrParseError,
            message: error.to_string(),
        }
    }
}

