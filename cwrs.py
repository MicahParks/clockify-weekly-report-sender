#!/usr/bin/env python
"""
[WHEN TO USE THIS FILE]
[INSTRUCTIONS FOR USING THIS FILE]

Project name: [MISSING]
Author: Micah Parks

This lives on the web at: [MISSING URL]
Target environment: python 3.7
"""

# Start standard library imports.
from os import getcwd, listdir, mkdir
from shutil import rmtree
from time import sleep
# End standard library imports.

# Start third party imports.
from selenium import webdriver
from selenium.webdriver.common.by import By
from selenium.webdriver.firefox.options import Options
from selenium.webdriver.firefox.webdriver import WebDriver
from selenium.webdriver.remote.webelement import WebElement
from selenium.webdriver.support import expected_conditions
from selenium.webdriver.support.ui import WebDriverWait
from pyvirtualdisplay import Display
# End third party imports.


MAX_WAIT_SEC_INT = 5
WEB_DRIVER_WAIT = None


def get_web_element(cssSelectorStr: str, webDriverObj: WebDriver) -> WebElement:
    """
    """
    web_driver_wait(cssSelectorStr=cssSelectorStr, webDriver=webDriverObj)
    elementObj = webDriverObj.find_element_by_css_selector(cssSelectorStr)
    webDriverObj.execute_script("arguments[0].scrollIntoView();", elementObj)
    return elementObj


def login_to_clockify(emailStr: str, firefoxWebDriver: WebDriver, passwordStr: str, urlStr: str) -> None:
    """
    """
    firefoxWebDriver.get(urlStr)
    emailCssSelectorStr = '#email'
    get_web_element(cssSelectorStr=emailCssSelectorStr, webDriverObj=firefoxWebDriver).send_keys(emailStr)
    passwordCssSelector = '#password'
    get_web_element(cssSelectorStr=passwordCssSelector, webDriverObj=firefoxWebDriver).send_keys(passwordStr + '\ue007')


def main() -> None:
    """
    The logic of the file.
    """
    display = Display(visible=0, size=(800, 600))
    display.start()
    downloadDirPathStr = getcwd() + '/tempdownloadz'
    options = Options()
    options.headless = True
    options.set_preference('browser.download.folderList', 2)
    options.set_preference('browser.download.manager.showWhenStarting', False)
    options.set_preference('browser.download.dir', downloadDirPathStr)
    options.set_preference('browser.helperApps.neverAsk.saveToDisk', 'application/pdf')
    options.set_preference('pdfjs.disabled', True)
    firefoxWebDriver = webdriver.Firefox(options=options)
    urlStr = 'https://clockify.me/login'
    login_to_clockify(emailStr='', firefoxWebDriver=firefoxWebDriver, passwordStr='', urlStr=urlStr)
    WebDriverWait(firefoxWebDriver, MAX_WAIT_SEC_INT).until(expected_conditions.url_changes(urlStr))
    try:
        rmtree(downloadDirPathStr)
    except FileNotFoundError:
        pass
    mkdir(downloadDirPathStr)
    download_detailed_report(firefoxWebDriver=firefoxWebDriver)
    while listdir(downloadDirPathStr) == 0:
        sleep(.1)
    firefoxWebDriver.close()
    # Email thing here
    display.stop()


def download_detailed_report(firefoxWebDriver: WebDriver) -> None:
    """
    """
    firefoxWebDriver.get('https://clockify.me/reports/detailed')
    printIconCssSelector = 'span.report-actions__item:nth-child(3)'
    get_web_element(cssSelectorStr=printIconCssSelector, webDriverObj=firefoxWebDriver).click()


def web_driver_wait(cssSelectorStr: str, webDriver: WebDriver) -> None:
    """
    Using "expected_conditions" you could wait for stuff to be clickable/usable by keyboard?
    """
    global WEB_DRIVER_WAIT
    if WEB_DRIVER_WAIT is None:
        WEB_DRIVER_WAIT = WebDriverWait(webDriver, MAX_WAIT_SEC_INT)
    WEB_DRIVER_WAIT.until(expected_conditions.presence_of_element_located((By.CSS_SELECTOR, cssSelectorStr)))


if __name__ == '__main__':
    main()
