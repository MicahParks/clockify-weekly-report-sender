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
from email.mime.application import MIMEApplication
from email.mime.multipart import MIMEMultipart
from email.mime.text import MIMEText
from email.utils import formatdate
from json import load
from os import getcwd, listdir, mkdir
from shutil import rmtree
from smtplib import SMTP
from time import sleep
# End standard library imports.

# Start third party imports.
from pyvirtualdisplay import Display
from selenium import webdriver
from selenium.common.exceptions import TimeoutException
from selenium.webdriver.common.by import By
from selenium.webdriver.firefox.options import Options
from selenium.webdriver.firefox.webdriver import WebDriver
from selenium.webdriver.remote.webelement import WebElement
from selenium.webdriver.support import expected_conditions
from selenium.webdriver.support.ui import WebDriverWait
# End third party imports.


MAX_WAIT_SEC_INT = 5
WEB_DRIVER_WAIT = None


def date_str(dateStr: str) -> str:
    """"
    """
    dateStrList = dateStr.split('_')
    yearStr = dateStrList[2]
    monthStr = dateStrList[0]
    dayStr = dateStrList[1]
    return yearStr + monthStr + dayStr


def download_detailed_report(firefoxWebDriver: WebDriver) -> str:
    """
    """
    firefoxWebDriver.get('https://clockify.me/reports/detailed')
    previousWeekCssSelectorStr = 'button.cl-btn:nth-child(2)'
    WebDriverWait(firefoxWebDriver, 5).until(
        expected_conditions.invisibility_of_element((By.CLASS_NAME, 'rotating-loader-wrapper')))
    get_web_element(cssSelectorStr=previousWeekCssSelectorStr, webDriver=firefoxWebDriver).click()
    printIconCssSelector = 'span.report-actions__item:nth-child(3)'
    WebDriverWait(firefoxWebDriver, 5).until(
        expected_conditions.invisibility_of_element((By.CLASS_NAME, 'rotating-loader-wrapper')))
    sleep(2)
    get_web_element(cssSelectorStr=printIconCssSelector, webDriver=firefoxWebDriver).click()
    try:
        billStr = get_web_element(cssSelectorStr='.report__total-time--third', webDriver=firefoxWebDriver).text.strip(
            '(USD) ')
        billStr = '$' + billStr
    except TimeoutException:
        billStr = '(website bug occurred)'
    return billStr


def email_weekly_report(bodyStr: str, ccStr: str, fromEmailStr: str, fromEmailPasswordStr: str, pdfPathStr: str,
                        subjectStr: str, toEmailStr: str, emailHostAddress: str = 'smtp.gmail.com',
                        pdfAttachmentNameStr: str = None, portInt: int = 587) -> None:
    """
    """
    if pdfAttachmentNameStr is None:
        pdfAttachmentNameStr = pdfPathStr.split('/')[-1]
    mimeMultipart = MIMEMultipart()
    mimeMultipart['Cc'] = ccStr
    mimeMultipart['Date'] = formatdate(localtime=True)
    mimeMultipart['From'] = fromEmailStr
    mimeMultipart['Subject'] = subjectStr
    mimeMultipart['To'] = toEmailStr
    mimeMultipart.attach(MIMEText(bodyStr))
    with open(pdfPathStr, 'rb') as inFile:
        pdfMimeApplication = MIMEApplication(inFile.read(), Name=pdfAttachmentNameStr)
    pdfMimeApplication['Content-Disposition'] = 'attachment; filename="{}"'.format(pdfAttachmentNameStr)
    mimeMultipart.attach(pdfMimeApplication)
    try:
        smtpSsl = SMTP(host=emailHostAddress, port=portInt)
        smtpSsl.ehlo()
        smtpSsl.starttls()
        smtpSsl.ehlo()
    except Exception as exceptionStr:
        print('Failure to connect to "{}:{}".\nException: "{}".'.format(emailHostAddress, portInt, exceptionStr))
        return
    try:
        smtpSsl.login(user=fromEmailStr, password=fromEmailPasswordStr)
    except Exception as exceptionStr:
        print('Failure to log into "{}" at "{}:{}"\nException: "{}".'.format(fromEmailStr, emailHostAddress, portInt,
                                                                             exceptionStr))
    smtpSsl.sendmail(fromEmailStr, toEmailStr, mimeMultipart.as_string())
    smtpSsl.close()


def get_pdf_date_str(pdfPathStr: str) -> str:
    """
    Clockify_Detailed_Report_09_03_2019-09_09_2019.pdf
    """
    pdfDateStr = pdfPathStr.split('/')[-1]
    pdfDateStr = pdfDateStr.lstrip('Clockify_Detailed_Report_')
    pdfDateStr = pdfDateStr.rstrip('.pdf')
    dateStrList = pdfDateStr.split('-')
    startDateStr = date_str(dateStr=dateStrList[0])
    endDateStr = date_str(dateStr=dateStrList[-1])
    pdfDateStr = startDateStr + '-' + endDateStr
    return pdfDateStr


def get_web_element(cssSelectorStr: str, webDriver: WebDriver) -> WebElement:
    """
    """
    web_driver_wait(cssSelectorStr=cssSelectorStr, webDriver=webDriver)
    element = webDriver.find_element_by_css_selector(cssSelectorStr)
    webDriver.execute_script("arguments[0].scrollIntoView();", element)
    return element


def login_to_clockify(emailStr: str, firefoxWebDriver: WebDriver, passwordStr: str, urlStr: str) -> None:
    """
    """
    firefoxWebDriver.get(urlStr)
    emailCssSelectorStr = '#email'
    get_web_element(cssSelectorStr=emailCssSelectorStr, webDriver=firefoxWebDriver).send_keys(emailStr)
    passwordCssSelector = '#password'
    get_web_element(cssSelectorStr=passwordCssSelector, webDriver=firefoxWebDriver).send_keys(passwordStr + '\ue007')


def main(jsonDict) -> None:
    """
    The logic of the file.
    """
    # display = Display(visible=0, size=(800, 600))
    # display.start()
    downloadDirPathStr = getcwd() + '/tempdownloadz'
    options = Options()
    # options.headless = True
    options.set_preference('browser.download.folderList', 2)
    options.set_preference('browser.download.manager.showWhenStarting', False)
    options.set_preference('browser.download.dir', downloadDirPathStr)
    options.set_preference('browser.helperApps.neverAsk.saveToDisk', 'application/pdf')
    options.set_preference('pdfjs.disabled', True)
    firefoxWebDriver = webdriver.Firefox(options=options)
    urlStr = 'https://clockify.me/login'
    clockifyEmailStr = jsonDict['clockifyEmailStr']
    clockifyPasswordStr = jsonDict['clockifyPasswordStr']
    login_to_clockify(emailStr=clockifyEmailStr, firefoxWebDriver=firefoxWebDriver, passwordStr=clockifyPasswordStr,
                      urlStr=urlStr)
    WebDriverWait(firefoxWebDriver, MAX_WAIT_SEC_INT).until(expected_conditions.url_changes(urlStr))
    try:
        rmtree(downloadDirPathStr)
    except FileNotFoundError:
        pass
    mkdir(downloadDirPathStr)
    billStr = download_detailed_report(firefoxWebDriver=firefoxWebDriver)
    while len(listdir(downloadDirPathStr)) == 0:
        sleep(.1)
    firefoxWebDriver.close()
    pdfPathStr = downloadDirPathStr + '/' + listdir(downloadDirPathStr)[-1]
    pdfDateStr = get_pdf_date_str(pdfPathStr=pdfPathStr)
    bodyStr = jsonDict['bodyStr'].format(pdfDateStr, billStr)
    ccStr = jsonDict['ccStr']
    fromEmailPasswordStr = jsonDict['fromEmailPasswordStr']
    fromEmailStr = jsonDict['fromEmailStr']
    pdfAttachmentNameStr = jsonDict['pdfAttachmentNameStr'].format(pdfDateStr)
    subjectStr = jsonDict['subjectStr']
    toEmailStr = jsonDict['toEmailStr'].format(pdfDateStr)
    email_weekly_report(bodyStr=bodyStr, ccStr=ccStr, fromEmailStr=fromEmailStr,
                        fromEmailPasswordStr=fromEmailPasswordStr,
                        pdfAttachmentNameStr=pdfAttachmentNameStr, pdfPathStr=pdfPathStr, subjectStr=subjectStr,
                        toEmailStr=toEmailStr)
    # display.stop()


def web_driver_wait(cssSelectorStr: str, webDriver: WebDriver) -> None:
    """
    Using "expected_conditions" you could wait for stuff to be clickable/usable by keyboard?
    """
    global WEB_DRIVER_WAIT
    if WEB_DRIVER_WAIT is None:
        WEB_DRIVER_WAIT = WebDriverWait(webDriver, MAX_WAIT_SEC_INT)
    WEB_DRIVER_WAIT.until(expected_conditions.presence_of_element_located((By.CSS_SELECTOR, cssSelectorStr)))


if __name__ == '__main__':
    with open('cwrs.json') as IN_FILE:
        JSON_DICT = load(IN_FILE)
    main(jsonDict=JSON_DICT)
