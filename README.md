# bctx
bctx is a tool for packing and uploading docker build context.

## Installation
```
go get github.com/orisano/bctx/cmd/bctx
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

## Author
Nao YONASHIRO (@orisano)

## License
MIT
