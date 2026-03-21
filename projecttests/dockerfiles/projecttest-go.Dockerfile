# Thin project overlay image for projecttests.
#
# Build with:
#   docker build -f projecttests/dockerfiles/projecttest-go.Dockerfile \
#     --build-arg BASE_IMAGE=omes-projecttest-go-base:latest \
#     --build-arg TEST_PROJECT=projecttests/go/tests/helloworld \
#     -t omes-projecttest-helloworld:latest .

ARG BASE_IMAGE
FROM ${BASE_IMAGE}

WORKDIR /app

ARG TEST_PROJECT
COPY ${TEST_PROJECT} /app/${TEST_PROJECT}

ENV TEST_PROJECT_PATH=/app/${TEST_PROJECT}
