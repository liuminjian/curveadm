FROM golang:1.20.11 as builder

#  install musl
RUN  wget -O /tmp/musl-1.2.4.tar.gz http://www.musl-libc.org/releases/musl-1.2.4.tar.gz \
    && tar -xvf /tmp/musl-1.2.4.tar.gz -C /tmp \
    && cd /tmp/musl-1.2.4 \
    && ./configure --prefix=/usr/local/  \
    && make \
    && make install \
    && cd - \
    && rm -rf /tmp/musl-1.2.4 /tmp/musl-1.2.4.tar.gz \
    ;

WORKDIR /go/src/
COPY . /go/src/

RUN make build-mod

# Final Stage
FROM debian:10
USER root:root
WORKDIR /root

ENV g_curveadm_home="/root/.curveadm"
ENV g_bin_dir="$g_curveadm_home/bin"
ENV PATH=$PATH:$g_bin_dir

RUN  mkdir -p $g_curveadm_home/{bin,data,plugins,logs,temp}


COPY --from=builder /go/src/configs/curveadm.cfg $g_curveadm_home/curveadm.cfg
COPY --from=builder /go/src/bin/curveadm $g_bin_dir/curveadm
COPY --from=builder /go/src/bin/pigeon $g_bin_dir/pigeon

RUN  chmod 755 "$g_bin_dir/curveadm" && chmod 755 "$g_bin_dir/pigeon"

ENTRYPOINT ["curveadm"]
