import time
from prettyprinter import cpprint

from pyftdi.spi import SpiController, SpiPort
from pyftdi.usbtools import UsbTools
from pyftdi.ftdi import Ftdi

status_register = 0x00
status_payload = 0x07
status_reset = 0x01


def get_device_url():
    utools = UsbTools()

    devices = utools.list_devices(
        urlstr="ftdi:///?",
        vdict={"ftdi": 0x403},
        pdict=Ftdi.PRODUCT_IDS,
        default_vendor=0x403,
    )

    if not devices:
        raise ValueError("No FTDI device found")

    device_urls = utools.build_dev_strings(
        scheme="ftdi",
        vdict={"ftdi": 0x403},
        pdict=Ftdi.PRODUCT_IDS,
        devdescs=devices,
    )

    found_valid_device = True

    for url in device_urls:
        if "???" not in url and "ftdi" in url:
            found_valid_device = True

    if not found_valid_device:
        raise ValueError("No valid FTDI device found when building URLs")

    return device_urls[0][0]




class PGA(int):
    def __new__(cls, value: int = 1):
        if value not in [1, 2, 4, 8, 16, 32, 64]:
            raise ValueError(f"Invalid PGA value: {value}")
        return super().__new__(cls, value)

    def __str__(self):
        return 

def flush(spi_port: SpiPort, verbose=True) -> None:
    if verbose:
        print("FLSH")
    spi_port.flush()
    time.sleep(0.001000)


def write(spi_port: SpiPort, data: bytes | int, verbose=True) -> None:
    for x in data:
        # noinspection StrFormat
        if verbose:
            # noinspection StrFormat
            print("\t->", ''.join(format(int(x), '08b')))
        try:
            spi_port.write(bytes(x))
        except Exception as e:
            raise e
        time.sleep(0.000100)

    time.sleep(0.001000)

    try:
        flush(spi_port, verbose)
    except Exception as e:
        raise e


# noinspection StrFormat
def read_reg(spi_port: SpiPort, register: int, expected: int = 3, verbose=True) -> bytes:
    print(f"RREG {register}")

    try:
        write(spi_port=spi_port, data=bytes([0x10 | register, 0x00]), verbose=verbose)
    except Exception as e:
        raise e

    time.sleep(0.001000)

    result = spi_port.read(expected)

    if not result:
        raise ValueError(f'no data read from register {register}')

    return result


def print_reg(spi_port: SpiPort, register: int, expected: int = 3, title: str = '') -> None:
    try:
        result = read_reg(spi_port, register, expected)
    except Exception as e:
        raise e

    if not result:
        raise ValueError("no data read from SPI port")

    if title:
        print(title)

    for x in result:
        # noinspection StrFormat
        print("\t<-", ''.join(format(x, '08b')))

    time.sleep(0.001000)


def write_reg(spi_port: SpiPort, register: int, payload: bytes | None | int, expected: int = 1, verbose=True) -> bytes:
    if verbose:
        print(f"WREG {register}")

    write(spi_port=spi_port, data=bytes([0x50 | register, 0x00, payload]))

    time.sleep(0.001000)

    result = spi_port.read(expected)

    if not result:
        raise ValueError("Status: No data read from SPI port")

    return result


def set_status(spi_port: SpiPort, verbose=True) -> None:
    print("Setting status")

    res = write_reg(spi_port, status_register, status_payload, 3, verbose)

    if not res:
        raise ValueError("Status: No data read from SPI port")

    if verbose:
        for x in res:
            # noinspection StrFormat
            print("\t<-", ''.join(format(int(x), '08b')))

    time.sleep(0.001000)


def init_adcon(spi_port: SpiPort, verbose=True) -> None:
    print("adcon init")

    try:
        adc = read_reg(spi_port, 0x02, 3, verbose)
    except Exception as e:
        raise e

    if not adc:
        raise ValueError("failed to initialize ADCON: got no data")

    time.sleep(0.001000)

    new_adc = (adc & 0x07) | PGA(32)

    adc = write_reg(spi_port, 0x02, 3, verbose)

def __main__():
    try:
        device_url = get_device_url()
    except Exception as e:
        print("Fatal: ", e)
        raise e

    if not device_url:
        print("No device URL found, but no error was raised")
        exit(1)

    print("Device URL: ", device_url)

    spi = SpiController()
    cfg = {}
    cfg.setdefault('frequency', 1700000)
    cfg.setdefault('turbo', 1)
    cfg.setdefault('debug', True)

    cpprint(cfg)

    try:
        spi.configure(device_url, **cfg)
        spi.flush()
        spi_port = spi.get_port(0)
        gpio_port = spi.get_gpio()
        gpio_port.set_direction()

        print("SPI CS: ", spi_port.cs)
        print("SPI Frequency: ", spi_port.frequency)

        set_status(spi_port)
        print_reg(spi_port, status_register, 1, "Status Register")
    except Exception as e:
        print("Fatal: ", e)
        raise e
    finally:
        print("Closing SPI port")
        spi.close()

if __name__ == "__main__":
    __main__()
