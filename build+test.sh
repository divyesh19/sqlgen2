#!/bin/bash -e
cd $(dirname $0)

. go-get.sh

PATH=$HOME/gopath/bin:$GOPATH/bin:$PATH

# delete artefacts from previous build (if any)
rm -f *.out */*.txt demo/*_sql.go

### Dependencies ###

go_get bitbucket.org/pkg/inflect bitbucket.org/pkg/inflect
go_get github.com/acsellers/inflections github.com/acsellers/inflections
go_get github.com/kortschak/utter github.com/kortschak/utter
go_get gopkg.in/yaml.v2 gopkg.in/yaml.v2

if ! type -p goveralls; then
  echo go get github.com/mattn/goveralls
  go get github.com/mattn/goveralls
fi

### Collection Types ###
# these generated files hardly ever need to change (see github.com/rickb777/runtemplate to do so)
[ -f model/type_set.go ]         || runtemplate -tpl simple/set.tpl -output model/type_set.go         Type=Type
[ -f sqlgen/code/string_set.go ] || runtemplate -tpl simple/set.tpl -output sqlgen/code/string_set.go Type=string
[ -f support/int64_set.go ]      || runtemplate -tpl simple/set.tpl -output support/int64_set.go      Type=int64

### Build Phase 1 ###

cd sqlgen
go install .

for d in code output parse; do
  echo sqlgen/$d...
  go test $1 -covermode=count -coverprofile=../$d.out ./$d
  go tool cover -func=../$d.out
  [ -z "$COVERALLS_TOKEN" ] || goveralls -coverprofile=$d.out -service=travis-ci -repotoken $COVERALLS_TOKEN
done

cd ..

### Build Phase 2 ###

go install .

for d in require schema sqlgen where; do
  echo ./$d...
  go test $1 -covermode=count -coverprofile=./$d.out ./$d
  go tool cover -func=./$d.out
  [ -z "$COVERALLS_TOKEN" ] || goveralls -coverprofile=$d.out -service=travis-ci -repotoken $COVERALLS_TOKEN
done

#echo .
#go test . -covermode=count -coverprofile=dot.out .
#go tool cover -func=dot.out
#[ -z "$COVERALLS_TOKEN" ] || goveralls -coverprofile=$d.out -service=travis-ci -repotoken $COVERALLS_TOKEN

### Demo ###

cd demo
./build.sh
