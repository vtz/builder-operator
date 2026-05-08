# CAN message handler drops frames under high bus load

## Labels
`bug`, `mcu`

## Description

When the CAN bus is under sustained high load (>80% utilization), the body-ecu
MCU firmware drops incoming CAN frames. The Zephyr CAN RX callback processes
each frame synchronously in the ISR context, which causes the hardware FIFO to
overflow before the next frame can be dequeued.

## Steps to Reproduce

1. Flash the body-ecu firmware to the nucleo_h755zi_q board.
2. Connect to a CAN bus with a traffic generator sending at >80% bus load.
3. Monitor the `can_rx_count` and `can_rx_drop` counters via the debug shell.
4. After 60 seconds, `can_rx_drop` is non-zero and growing.

## Expected Behavior

Zero dropped frames under sustained load. The RX path should offload frame
processing to a work queue so the ISR returns immediately and the hardware FIFO
does not overflow.

## Affected Files

- `src/app/src/can_handler.c` -- CAN RX callback and frame dispatch
- `src/app/src/signal_bus.c` -- downstream signal processing (called from RX callback)

## Acceptance Criteria

- [ ] CAN RX callback enqueues frames to a `k_msgq` or `k_fifo` instead of processing inline
- [ ] A dedicated work queue thread dequeues and dispatches frames
- [ ] `can_rx_drop` remains zero after 5 minutes at 90% bus load
- [ ] Firmware builds cleanly for `nucleo_h755zi_q` via `west build`
- [ ] Verified on hardware via Jumpstarter flash + serial console check
