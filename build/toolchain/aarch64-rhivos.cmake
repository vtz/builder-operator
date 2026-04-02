# =============================================================================
# CMake Toolchain: AArch64 RHIVOS (Red Hat In-Vehicle OS)
# Cross-compilation from x86_64 host to AArch64 target
# =============================================================================

set(CMAKE_SYSTEM_NAME Linux)
set(CMAKE_SYSTEM_PROCESSOR aarch64)

# --- Cross-compiler ---
set(CMAKE_C_COMPILER   aarch64-linux-gnu-gcc)
set(CMAKE_CXX_COMPILER aarch64-linux-gnu-g++)
set(CMAKE_AR           aarch64-linux-gnu-ar)
set(CMAKE_RANLIB       aarch64-linux-gnu-ranlib)
set(CMAKE_STRIP        aarch64-linux-gnu-strip)
set(CMAKE_OBJCOPY      aarch64-linux-gnu-objcopy)

# --- Sysroot (set via environment or default path) ---
if(DEFINED ENV{RHIVOS_SYSROOT})
  set(CMAKE_SYSROOT $ENV{RHIVOS_SYSROOT})
else()
  set(CMAKE_SYSROOT /opt/rhivos-sysroot/aarch64)
endif()

# --- Architecture flags ---
set(CMAKE_C_FLAGS_INIT   "-march=armv8-a -mtune=cortex-a76")
set(CMAKE_CXX_FLAGS_INIT "-march=armv8-a -mtune=cortex-a76")

# --- Hardening flags (ISO 26262 Part 6 / ISO 21434 Clause 10) ---
set(HARDENING_FLAGS "-Wall -Wextra -Werror \
  -fstack-protector-strong -D_FORTIFY_SOURCE=2 \
  -fPIE -fstack-clash-protection \
  -Wformat -Wformat-security -Wformat=2 \
  -Wconversion -Wsign-conversion \
  -Wcast-align -Wshadow")

set(CMAKE_C_FLAGS   "${CMAKE_C_FLAGS_INIT} ${HARDENING_FLAGS}" CACHE STRING "" FORCE)
set(CMAKE_CXX_FLAGS "${CMAKE_CXX_FLAGS_INIT} ${HARDENING_FLAGS}" CACHE STRING "" FORCE)

set(CMAKE_EXE_LINKER_FLAGS_INIT "-pie -Wl,-z,relro -Wl,-z,now -Wl,-z,noexecstack")

# --- Search path configuration ---
set(CMAKE_FIND_ROOT_PATH ${CMAKE_SYSROOT})
set(CMAKE_FIND_ROOT_PATH_MODE_PROGRAM NEVER)
set(CMAKE_FIND_ROOT_PATH_MODE_LIBRARY ONLY)
set(CMAKE_FIND_ROOT_PATH_MODE_INCLUDE ONLY)
set(CMAKE_FIND_ROOT_PATH_MODE_PACKAGE ONLY)

# --- RHIVOS-specific definitions ---
add_definitions(-DPLATFORM_RHIVOS)
add_definitions(-DARCH_AARCH64)
add_definitions(-D_GNU_SOURCE)
