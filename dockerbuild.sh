#!/bin/bash

go build
docker build -t jotak/discomon .
rm discomon
