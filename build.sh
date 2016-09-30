#!/bin/bash
go-bindata assets/...
rm assets/styles/style.css
lessc assets/styles/src/index.less assets/styles/style.css
