# =============================================================================
# CMake Toolchain: ARM Cortex-M4 FreeRTOS
# Cross-compilation for bare-metal RTOS target
# =============================================================================

set(CMAKE_SYSTEM_NAME Generic)
set(CMAKE_SYSTEM_PROCESSOR arm)

# --- Cross-compiler (ARM Embedded Toolchain) ---
set(CMAKE_C_COMPILER   arm-none-eabi-gcc)
set(CMAKE_CXX_COMPILER arm-none-eabi-g++)
set(CMAKE_AR           arm-none-eabi-ar)
set(CMAKE_RANLIB       arm-none-eabi-ranlib)
set(CMAKE_OBJCOPY      arm-none-eabi-objcopy)
set(CMAKE_SIZE         arm-none-eabi-size)

# --- Disable shared libraries (bare-metal) ---
set(BUILD_SHARED_LIBS OFF)
set(CMAKE_SHARED_LIBRARY_LINK_C_FLAGS "")
set(CMAKE_SHARED_LIBRARY_LINK_CXX_FLAGS "")

# --- MCU-specific flags ---
set(MCU_FLAGS "-mcpu=cortex-m4 -mthumb -mfloat-abi=hard -mfpu=fpv4-sp-d16")

# --- Compiler flags ---
set(COMMON_FLAGS "${MCU_FLAGS} \
  -Wall -Wextra -Werror \
  -ffunction-sections -fdata-sections \
  -fno-common -fno-exceptions \
  -D_FORTIFY_SOURCE=1 \
  -Wformat -Wformat-security \
  -Wconversion -Wsign-conversion \
  -Wcast-align -Wshadow \
  -Wstrict-prototypes -Wmissing-prototypes")

set(CMAKE_C_FLAGS   "${COMMON_FLAGS} -std=c11" CACHE STRING "" FORCE)
set(CMAKE_CXX_FLAGS "${COMMON_FLAGS} -std=c++17 -fno-rtti" CACHE STRING "" FORCE)

# --- Linker flags ---
set(CMAKE_EXE_LINKER_FLAGS_INIT
  "${MCU_FLAGS} \
  --specs=nano.specs --specs=nosys.specs \
  -Wl,--gc-sections \
  -Wl,--print-memory-usage \
  -Wl,-Map=output.map"
  CACHE STRING "" FORCE)

# --- Linker script (project-specific) ---
if(DEFINED LINKER_SCRIPT)
  set(CMAKE_EXE_LINKER_FLAGS "${CMAKE_EXE_LINKER_FLAGS_INIT} -T${LINKER_SCRIPT}")
endif()

# --- Search paths ---
set(CMAKE_FIND_ROOT_PATH_MODE_PROGRAM NEVER)
set(CMAKE_FIND_ROOT_PATH_MODE_LIBRARY ONLY)
set(CMAKE_FIND_ROOT_PATH_MODE_INCLUDE ONLY)
set(CMAKE_FIND_ROOT_PATH_MODE_PACKAGE ONLY)

# --- Try compile configuration ---
set(CMAKE_TRY_COMPILE_TARGET_TYPE STATIC_LIBRARY)

# --- RTOS-specific definitions ---
add_definitions(-DPLATFORM_RTOS)
add_definitions(-DARCH_CORTEX_M4)
add_definitions(-DUSE_FREERTOS)
add_definitions(-DUSE_HAL_DRIVER)
