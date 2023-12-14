# rustup install nightly
# cargo +nightly build --release
cargo build --release

# judge dynamic lib file by OS
if [[ "$(uname)" == "Darwin" ]]; then
    dynamic_lib_file="libxcrypto.dylib"
else
    dynamic_lib_file="libxcrypto.so"
fi

cp ./target/release/${dynamic_lib_file} lib/
# go build -o test_go -ldflags="-r ./lib" *.go