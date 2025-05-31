# P5 Tasks

## build

~~~sh
cargo build --release --bin p5
~~~

## check

~~~sh
cargo clippy
cargo fmt --all -- --check
~~~

## fix

~~~sh
cargo clippy --fix
cargo fmt --all
~~~

## test

### build
~~~sh
docker build . -f Dockerfile.test -t p5:tests
~~~

### unit

~~~sh
cargo test --workspace --lib --bins
~~~

### integration

~~~sh
set -e
mask test build
docker run --rm -v "$(pwd)/target:/app/target" p5:tests "*integration"
~~~

### debug
~~~sh
set -e
mask test build
docker run -it --rm -v "$(pwd)/target:/app/target" p5:tests "debug"
~~~
