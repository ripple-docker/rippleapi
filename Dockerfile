FROM golang:1.12-alpine

WORKDIR /go/src/zxq.co/ripple/rippleapi
COPY . .

RUN go get -d -v ./...
RUN CGO_ENABLED=0 go install -v ./...

FROM alpine
WORKDIR /rippleapi/
COPY --from=0 /go/bin/rippleapi ./

# Agree to License
RUN mkdir ~/.config && touch ~/.config/ripple_license_agreed

# generate config
RUN ./rippleapi

COPY ./entrypoint.sh .
RUN chmod +x entrypoint.sh

ENV OSUAPIKEY pleaseuseme
ENV MYSQL_ROOT_PASSWORD changeme
ENV APISECRET Potato
EXPOSE 40001

ENTRYPOINT [ "./entrypoint.sh" ]
CMD ["./rippleapi"]
