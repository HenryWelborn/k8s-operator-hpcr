# Copyright 2023 IBM Corp.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#	http:#www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.package datasource

FROM registry.access.redhat.com/ubi8/go-toolset as build_layer

COPY . /src

RUN cd /src && \
    mkdir -p /opt/app-root/bin && \
    go build -ldflags "-X github.com/ibm-hyper-protect/k8s-operator-hpcr/cli.compiled=$(date +%s) -s -w" -o /opt/app-root/bin/k8s-operator-hpcr main.go && \
    ldd /opt/app-root/bin/k8s-operator-hpcr | tr -s '[:blank:]' '\n' | grep '^/' | xargs -I % sh -c 'mkdir -p $(dirname /opt/app-root/bin%); cp % /opt/app-root/bin%;'    

FROM registry.access.redhat.com/ubi8/ubi-minimal as base_layer

FROM scratch

COPY --from=base_layer /etc/ssl/certs/ /etc/ssl/certs/
COPY --from=base_layer /etc/pki/tls/ /etc/pki/tls/
COPY --from=base_layer /etc/pki/ca-trust/ /etc/pki/ca-trust/

COPY --from=build_layer /opt/app-root/bin /

EXPOSE 8080

ENTRYPOINT [ "/k8s-operator-hpcr" ]
