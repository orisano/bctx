# bctx
bctx is a tool for packing and uploading docker build context.

## Installation
```
go get github.com/orisano/bctx/cmd/bctx
```
or
```
curl -o /usr/local/bin/bctx -SsL $(curl -s https://api.github.com/repos/orisano/bctx/releases/latest | jq -r '.assets[].browser_download_url' | grep darwin) && chmod +x /usr/local/bin/bctx
```

## How to use
```
$ bctx -help
  -dest string
    	destination path, supported gs://, s3:// and dir (required)
  -f string
    	override Dockerfile
  -ignore string
    	.dockerignore path (default "$src/.dockerignore")
  -src string
    	source directory (default ".")
```

To show files of the build context. (order by size)
```
$ bctx -dest - | tar tv | sort -k3 -r | less
```

## Author
Nao Yonashiro (@orisano)

## License
MIT
