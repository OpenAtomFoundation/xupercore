#include <stdio.h>
#include <sstream>
#include "xchain/xchain.h"

struct Memory : public xchain::Contract {};

DEFINE_METHOD(Memory, initialize) {
  xchain::Context* ctx = self.context();
  ctx->ok("ok");
}

DEFINE_METHOD(Memory, call_malloc) {
  xchain::Context* ctx = self.context();
  std::string countStr = ctx->arg("count");
  if (countStr.empty()) {
    ctx->error("missing count");
    return;
  }
        std::stringstream ss;
  int count = atoi(countStr.c_str());
  for (int i = 0; i < count; i++) {
    char* ptr = (char*)malloc(1024 * 64);
    if (ptr == NULL) {
    ss<<i;
      ctx->error(ss.str());
      return;
    }
    printf("%s\n", ptr);
  }
  ctx->ok("ok");
}
