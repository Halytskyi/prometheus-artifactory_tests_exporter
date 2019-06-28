FROM golang:1.11.5 as builder
ENV PACKAGE_PATH $GOPATH/src/git.host/mypackage
RUN mkdir -p  $PACKAGE_PATH
COPY . $PACKAGE_PATH
WORKDIR $PACKAGE_PATH
ARG version_string
ARG binary_name
ENV BINARY_NAME $binary_name
RUN make build && cp ${binary_name} /${binary_name}

FROM ruby:2.6
RUN  gem install --quiet --no-document fpm

ARG binary_name
ARG deb_package_name
ARG version_string
ARG deb_package_description
ARG pkg_vendor
ARG pkg_maintainer
ARG pkg_url

RUN mkdir /deb-packages
COPY --from=builder /$binary_name /deb-packages/usr/bin/$binary_name
COPY dpkg-sources/dirs /deb-packages
RUN mkdir dpkg-sources
COPY dpkg-sources /dpkg-sources/
WORKDIR dpkg-sources
RUN fpm --output-type deb \
  --input-type dir --chdir /deb-packages \
  --name $binary_name \
  --version $version_string \
  --description "${deb_package_description}" \
  --vendor "${pkg_vendor}" \
  --maintainer "${pkg_maintainer}" \
  --url "${pkg_url}" \
  --deb-systemd "startup/prometheus-artifactory-tests-exporter.service" \
  --deb-default "prometheus-artifactory-tests-exporter" \
  -p ${deb_package_name}-${version_string}.deb \
  etc usr && cp *.deb /deb-packages/
