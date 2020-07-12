FROM golang:1.14.6
ADD . /app
WORKDIR /app
RUN GOOS=linux GOARCH=amd64 go build cmd/phunter.go

FROM centos:7
RUN yum update -y
RUN yum groupinstall "Development Tools" -y
RUN git clone --recursive https://github.com/adsr/phpspy.git
RUN cd phpspy && make

FROM centos:7
ENV DOCKER_VERSION=19.03.11
RUN curl -fsSLO https://download.docker.com/linux/static/stable/x86_64/docker-${DOCKER_VERSION}.tgz \
  && tar xzvf docker-${DOCKER_VERSION}.tgz --strip 1 \
  -C /usr/local/bin docker/docker \
  && rm docker-${DOCKER_VERSION}.tgz

COPY --from=0 /app/phunter .
COPY --from=1 /phpspy .
RUN ln -s /phpspy /usr/bin/phpspy

ENTRYPOINT ["/phunter"]
