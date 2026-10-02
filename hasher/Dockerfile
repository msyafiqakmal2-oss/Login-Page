FROM rust:1.85 AS build
WORKDIR /src
COPY hasher/ .
RUN cargo build --release

FROM debian:bookworm-slim
COPY --from=build /src/target/release/hasher /hasher
EXPOSE 8081
CMD ["/hasher"]
