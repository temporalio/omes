# Thin project overlay image for workflowtests.
#
# Build with:
#   docker build -f workflowtests/dockerfiles/workflowtest-project.Dockerfile \
#     --build-arg BASE_IMAGE=omes-workflowtest-go-base:latest \
#     --build-arg TEST_PROJECT=workflowtests/go/tests/simpletest \
#     -t omes-workflowtest-go-simpletest:latest .
#
#   docker build -f workflowtests/dockerfiles/workflowtest-project.Dockerfile \
#     --build-arg BASE_IMAGE=omes-workflowtest-python-base:latest \
#     --build-arg TEST_PROJECT=workflowtests/python/tests/simple_test \
#     -t omes-workflowtest-python-simple-test:latest .
#
#   docker build -f workflowtests/dockerfiles/workflowtest-project.Dockerfile \
#     --build-arg BASE_IMAGE=omes-workflowtest-typescript-base:latest \
#     --build-arg TEST_PROJECT=workflowtests/typescript/tests/simple-test \
#     -t omes-workflowtest-typescript-simple-test:latest .

ARG BASE_IMAGE
FROM ${BASE_IMAGE}

WORKDIR /app

ARG TEST_PROJECT
COPY ${TEST_PROJECT} /app/${TEST_PROJECT}

ENV TEST_PROJECT_PATH=/app/${TEST_PROJECT}
