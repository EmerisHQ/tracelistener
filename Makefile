OBJS = $(shell find cmd -type d  -mindepth 1 -execdir printf '%s\n' {} +)
BASEPKG = github.com/allinbits/demeris-backend
EXTRAFLAGS :=

.PHONY: $(OBJS) clean generate-swagger

all: $(OBJS)

clean:
	@rm -rf build docs/swagger.* docs/docs.go

generate-swagger:
	go generate ${BASEPKG}/docs
	@rm docs/docs.go

ifdef DEBUG
$(OBJS): DEBUG_LDFLAGS =
else
$(OBJS): DEBUG_LDFLAGS = -ldflags='-s -w'
endif
$(OBJS):
	go build -o build/$@ $(DEBUG_LDFLAGS) ${EXTRAFLAGS} ${BASEPKG}/cmd/$@
