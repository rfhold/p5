# P5 Tasks

## tools
~~~sh
go install github.com/charmbracelet/vhs@latest
~~~

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
docker build . -f Dockerfile -t p5:tests --target test
~~~

### unit

~~~sh
cargo test --workspace --lib --bins
~~~

### integration

~~~sh
set -e
mask test build
docker run --rm p5:tests "*integration"
~~~

### debug
~~~sh
set -e
mask test build
docker run -it --rm p5:tests "debug"
~~~

## vhs

### build
~~~sh
docker build . -f Dockerfile -t p5:vhs --target vhs
~~~

### run
~~~sh
mask vhs build
docker run -it --rm -v "$PWD/tapes/output:/app/tapes/output" p5:vhs
~~~
