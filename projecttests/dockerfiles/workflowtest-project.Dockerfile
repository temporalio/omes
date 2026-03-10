# Thin project overlay image for projecttests.
#
# Build with:
#   docker build -f projecttests/dockerfiles/workflowtest-project.Dockerfile \
#     --build-arg BASE_IMAGE=omes-workflowtest-go-base:latest \
#     --build-arg TEST_PROJECT=projecttests/go/tests/helloworld \
#     -t omes-workflowtest-go-helloworld:latest .

ARG BASE_IMAGE
FROM ${BASE_IMAGE}

WORKDIR /app

ARG TEST_PROJECT
COPY ${TEST_PROJECT} /app/${TEST_PROJECT}

ENV TEST_PROJECT_PATH=/app/${TEST_PROJECT}
