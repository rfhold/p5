# Pulumi Automation Tasks

## build

~~~sh
cargo build --release
~~~

## check

~~~sh
cargo clippy
cargo fmt --all -- --check
~~~

## test

### unit

~~~sh
cargo test
~~~

### integration

~~~sh
cargo test --test integration
~~~
