# =============================================================================
# RHIVOS/RTOS Edge Application Build System
# Supports: rhivos-aarch64, rhivos-x86_64, rtos-cortex-m4
# Standards: ISO 21434, ISO 26262, IEC 62443, ASPICE
# =============================================================================

# --- Platform Configuration ---
PLATFORM       ?= rhivos-aarch64
PROJECT        := edge-app
VERSION        := 0.1.0
BUILD_DIR      := build/out
SRC_DIR        := src
TEST_DIR       := test
FUZZ_DIR       := fuzz
COVERAGE_DIR   := coverage

# --- Toolchain Selection ---
ifeq ($(PLATFORM),rhivos-aarch64)
  CROSS_COMPILE  ?= aarch64-linux-gnu-
  CMAKE_TOOLCHAIN := build/toolchain/aarch64-rhivos.cmake
  ARCH_FLAGS     := -march=armv8-a
  DEPLOY_METHOD  := ostree
endif

ifeq ($(PLATFORM),rhivos-x86_64)
  CROSS_COMPILE  ?=
  CMAKE_TOOLCHAIN :=
  ARCH_FLAGS     := -march=x86-64-v3
  DEPLOY_METHOD  := ostree
endif

ifeq ($(PLATFORM),rtos-cortex-m4)
  CROSS_COMPILE  ?= arm-none-eabi-
  CMAKE_TOOLCHAIN := build/toolchain/rtos-freertos.cmake
  ARCH_FLAGS     := -mcpu=cortex-m4 -mthumb -mfloat-abi=hard -mfpu=fpv4-sp-d16
  DEPLOY_METHOD  := firmware-ota
  LDFLAGS_EXTRA  := --specs=nano.specs --specs=nosys.specs -T linker/stm32f4.ld
endif

# --- Compiler Configuration ---
CC             := $(CROSS_COMPILE)gcc
CXX            := $(CROSS_COMPILE)g++
AR             := $(CROSS_COMPILE)ar
OBJCOPY        := $(CROSS_COMPILE)objcopy
SIZE           := $(CROSS_COMPILE)size

# --- Hardening Flags (ISO 26262 Part 6, MISRA C:2012) ---
COMMON_FLAGS   := -Wall -Wextra -Werror \
                  -Wformat -Wformat-security -Wformat=2 \
                  -Wconversion -Wsign-conversion \
                  -Wcast-align -Wcast-qual \
                  -Wswitch-default -Wswitch-enum \
                  -Wuninitialized -Winit-self \
                  -Wstrict-prototypes \
                  -Wmissing-prototypes -Wmissing-declarations \
                  -Wredundant-decls \
                  -Wnested-externs \
                  -Wshadow -Wundef \
                  -Wpointer-arith \
                  -Wno-unused-parameter \
                  -fstack-protector-strong \
                  -fno-common \
                  $(ARCH_FLAGS)

# Security hardening (enabled for Linux targets, not bare-metal RTOS)
ifneq ($(PLATFORM),rtos-cortex-m4)
  SECURITY_FLAGS := -D_FORTIFY_SOURCE=2 -fPIE -fstack-clash-protection
  LDFLAGS_SECURITY := -pie -Wl,-z,relro -Wl,-z,now -Wl,-z,noexecstack
else
  SECURITY_FLAGS := -D_FORTIFY_SOURCE=1 -fno-exceptions
  LDFLAGS_SECURITY :=
endif

CFLAGS         := -std=c11 $(COMMON_FLAGS) $(SECURITY_FLAGS) -O2
CXXFLAGS       := -std=c++17 $(COMMON_FLAGS) $(SECURITY_FLAGS) -O2
LDFLAGS        := $(LDFLAGS_SECURITY) $(LDFLAGS_EXTRA)

# --- Source Files ---
SRCS           := $(wildcard $(SRC_DIR)/*.c)
OBJS           := $(patsubst $(SRC_DIR)/%.c,$(BUILD_DIR)/%.o,$(SRCS))
TEST_SRCS      := $(wildcard $(TEST_DIR)/*.c)
TEST_OBJS      := $(patsubst $(TEST_DIR)/%.c,$(BUILD_DIR)/test/%.o,$(TEST_SRCS))

# --- Targets ---
.PHONY: all clean test lint sbom fuzz coverage cmake-build help

all: $(BUILD_DIR)/$(PROJECT)

help:
	@echo "RHIVOS/RTOS Edge Application Build System"
	@echo ""
	@echo "Usage: make [target] PLATFORM=<platform>"
	@echo ""
	@echo "Platforms:"
	@echo "  rhivos-aarch64  - RHIVOS on AArch64 (default)"
	@echo "  rhivos-x86_64   - RHIVOS on x86_64"
	@echo "  rtos-cortex-m4  - FreeRTOS on ARM Cortex-M4"
	@echo ""
	@echo "Targets:"
	@echo "  all       - Build the application (default)"
	@echo "  clean     - Remove build artifacts"
	@echo "  test      - Run unit tests (ASPICE SWE.4)"
	@echo "  lint      - Run static analysis (ISO 26262 Part 6)"
	@echo "  sbom      - Generate SBOM (ISO 21434 supply chain)"
	@echo "  fuzz      - Run fuzz testing (IEC 62443 VIT-C)"
	@echo "  coverage  - Generate code coverage report (ISO 26262 Table 12)"
	@echo "  cmake-build - Build using CMake with cross-compilation toolchain"
	@echo ""

# --- Build Rules ---
$(BUILD_DIR)/$(PROJECT): $(OBJS)
	@mkdir -p $(dir $@)
	$(CC) $(LDFLAGS) -o $@ $^
ifeq ($(PLATFORM),rtos-cortex-m4)
	$(OBJCOPY) -O binary $@ $@.bin
	$(OBJCOPY) -O ihex $@ $@.hex
	$(SIZE) $@
endif
	@echo "[BUILD] $(PLATFORM) -> $@"

$(BUILD_DIR)/%.o: $(SRC_DIR)/%.c
	@mkdir -p $(dir $@)
	$(CC) $(CFLAGS) -MMD -MP -c -o $@ $<

-include $(OBJS:.o=.d)

# --- CMake Cross-Compilation ---
cmake-build:
ifdef CMAKE_TOOLCHAIN
	cmake -B $(BUILD_DIR)/cmake \
	  -DCMAKE_TOOLCHAIN_FILE=$(CMAKE_TOOLCHAIN) \
	  -DCMAKE_BUILD_TYPE=Release \
	  -DPLATFORM=$(PLATFORM)
	cmake --build $(BUILD_DIR)/cmake --parallel
else
	cmake -B $(BUILD_DIR)/cmake -DCMAKE_BUILD_TYPE=Release
	cmake --build $(BUILD_DIR)/cmake --parallel
endif

# --- Unit Tests (ASPICE SWE.4 / ISO 26262 Part 6 Table 10) ---
test: CFLAGS += -DUNIT_TEST -g -O0 --coverage
test: LDFLAGS += --coverage
test: clean $(BUILD_DIR)/test/test_runner
	$(BUILD_DIR)/test/test_runner
	@echo "[TEST] Unit tests completed"

$(BUILD_DIR)/test/test_runner: $(TEST_OBJS) $(filter-out $(BUILD_DIR)/main.o, $(OBJS))
	@mkdir -p $(dir $@)
	$(CC) $(LDFLAGS) -o $@ $^

$(BUILD_DIR)/test/%.o: $(TEST_DIR)/%.c
	@mkdir -p $(dir $@)
	$(CC) $(CFLAGS) -I$(SRC_DIR) -MMD -MP -c -o $@ $<

# --- Code Coverage (ISO 26262 Part 6 Table 12) ---
# ASIL-A: Statement coverage
# ASIL-B: Branch coverage (decision coverage)
# ASIL-C/D: MC/DC coverage (requires specialized tools)
coverage: test
	@mkdir -p $(COVERAGE_DIR)
	gcov -b -c $(BUILD_DIR)/*.gcda -o $(BUILD_DIR)
	lcov --capture --directory $(BUILD_DIR) --output-file $(COVERAGE_DIR)/coverage.info \
	  --rc lcov_branch_coverage=1
	lcov --remove $(COVERAGE_DIR)/coverage.info '/usr/*' '$(TEST_DIR)/*' \
	  --output-file $(COVERAGE_DIR)/coverage.info --rc lcov_branch_coverage=1
	genhtml $(COVERAGE_DIR)/coverage.info --output-directory $(COVERAGE_DIR)/html \
	  --branch-coverage
	lcov --summary $(COVERAGE_DIR)/coverage.info --rc lcov_branch_coverage=1
	@echo "[COVERAGE] Report: $(COVERAGE_DIR)/html/index.html"
	@echo "[COVERAGE] Branch coverage required for ASIL-B+"

# --- Static Analysis / Linting (ISO 26262 Part 6, ISO 21434 Clause 10) ---
lint:
	@echo "[LINT] Running cppcheck..."
	cppcheck --enable=all --std=c11 --error-exitcode=1 \
	  --suppress=missingIncludeSystem \
	  --checkers-report=$(BUILD_DIR)/cppcheck-report.txt \
	  $(SRC_DIR)/
	@echo "[LINT] Running Semgrep with automotive rules..."
	semgrep --config security/semgrep.yml $(SRC_DIR)/ --sarif -o $(BUILD_DIR)/semgrep.sarif
	@echo "[LINT] Static analysis complete"

# --- SBOM Generation (ISO 21434 Supply Chain, IEC 62443-4-1) ---
sbom:
	@mkdir -p $(BUILD_DIR)/sbom
	syft dir:. -o cyclonedx-json=$(BUILD_DIR)/sbom/sbom.cyclonedx.json
	syft dir:. -o spdx-json=$(BUILD_DIR)/sbom/sbom.spdx.json
	@echo "[SBOM] CycloneDX: $(BUILD_DIR)/sbom/sbom.cyclonedx.json"
	@echo "[SBOM] SPDX:      $(BUILD_DIR)/sbom/sbom.spdx.json"

# --- Fuzz Testing (IEC 62443 VIT-C, ISO 21434 Vulnerability Testing) ---
fuzz: CC := clang
fuzz: CFLAGS += -fsanitize=fuzzer,address,undefined -g -O1 -DFUZZ_TEST
fuzz:
	@mkdir -p $(BUILD_DIR)/fuzz/corpus
	$(CC) $(CFLAGS) -I$(SRC_DIR) $(FUZZ_DIR)/fuzz_target.c $(SRCS) \
	  -o $(BUILD_DIR)/fuzz/fuzz_runner \
	  -fsanitize=fuzzer,address,undefined
	$(BUILD_DIR)/fuzz/fuzz_runner $(BUILD_DIR)/fuzz/corpus \
	  -max_total_time=300 -print_final_stats=1
	@echo "[FUZZ] Fuzzing complete. Corpus: $(BUILD_DIR)/fuzz/corpus"

# --- Clean ---
clean:
	rm -rf $(BUILD_DIR) $(COVERAGE_DIR)
	@echo "[CLEAN] Build artifacts removed"
