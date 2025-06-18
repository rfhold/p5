# P5 Tasks

## install
> Install P5
~~~sh
cargo install --path .
~~~

## tools
> Install the tools required to run the tasks in this file
~~~sh
go install github.com/charmbracelet/vhs@latest
~~~

## build
> Build the project
~~~sh
cargo build --release --bin p5
~~~

## publish
> Publish the packages to crates.io
### pulumi-automation
> Publish the `pulumi-automation` package
~~~sh
cargo publish --package pulumi-automation
~~~

### p5
> Publish the `p5` package
~~~sh
cargo publish --package p5
~~~

### all
> Publish all packages
~~~sh
mask publish pulumi-automation
mask publish p5
~~~

## check
> Check the code for errors and formatting
~~~sh
cargo clippy
cargo fmt --all -- --check
~~~

### all
> Check linting, formatting, run tests, and VHS
~~~sh
set -e
mask check
mask test unit
mask test integration
mask vhs run
~~~

## fix
> Fix the code formatting and linting issues
~~~sh
cargo clippy --fix
cargo fmt --all
~~~

## test
> Run the tests for the project
### build
> Build the integration test docker image
~~~sh
docker build . -f Dockerfile -t p5:tests --target test
~~~

### unit
> Run the unit tests for the project.
~~~sh
cargo test --workspace --lib --bins
~~~

### integration
> Run the integration tests for the project
~~~sh
set -e
mask test build
docker run --rm p5:tests "*integration"
~~~

### debug
> Run the debug tests for the project
~~~sh
set -e
mask test build
docker run -it --rm -v "$PWD/pulumi-automation/tests/fixtures/dumps:/app/pulumi-automation/tests/fixtures/dumps" p5:tests "debug"
~~~

## vhs
> Run VHS to record terminal sessions
### build
> Build the VHS docker image
~~~sh
docker build . -f Dockerfile -t p5:vhs --target vhs
~~~

### run
> Run the VHS sessions
~~~sh
set -e
mask vhs build
docker run -it --rm -v "$PWD/tapes/output:/app/tapes/output" p5:vhs
~~~
