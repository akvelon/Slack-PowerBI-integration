FROM chromedp/headless-shell


COPY --from=golang /usr/local/go/ /usr/local/go/
ENV PATH="/usr/local/go/bin:${PATH}"

RUN apt update -y
RUN apt install ca-certificates -y

RUN mkdir /app
COPY . /app 
WORKDIR /app

RUN go mod tidy 

EXPOSE 80
ENTRYPOINT [ "go", "run", "/app/reportengine.go" ]