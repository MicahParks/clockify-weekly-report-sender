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
from argparse import ArgumentParser, FileType
from datetime import datetime, timedelta
from email.mime.application import MIMEApplication
from email.mime.multipart import MIMEMultipart
from email.mime.text import MIMEText
from email.utils import formatdate
from json import dumps, load
from smtplib import SMTP
# End standard library imports.

# Start third party imports.
from requests import get, post
# End third party imports.


def email_weekly_report(bodyStr: str, ccStr: str, fromEmailStr: str, fromEmailPasswordStr: str, pdfFile: bytes,
                        subjectStr: str, toEmailsStrList: list, emailHostAddress: str = 'smtp.gmail.com',
                        pdfAttachmentNameStr: str = None, portInt: int = 587) -> None:
    """
    """
    mimeMultipart = MIMEMultipart()
    mimeMultipart['Cc'] = ccStr
    mimeMultipart['Date'] = formatdate(localtime=True)
    mimeMultipart['From'] = fromEmailStr
    mimeMultipart['Subject'] = subjectStr
    mimeMultipart['To'] = ', '.join(toEmailsStrList)
    mimeMultipart.attach(MIMEText(bodyStr))
    pdfMimeApplication = MIMEApplication(pdfFile, Name=pdfAttachmentNameStr)
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
    smtpSsl.sendmail(fromEmailStr, toEmailsStrList, mimeMultipart.as_string())
    smtpSsl.close()


def get_bill_str(data: dict, headers: dict, workspaceStr: str) -> str:
    """
    """
    response = post('https://global.api.clockify.me/workspaces/{}/reports/new/summary/'.format(workspaceStr),
                    headers=headers, data=dumps(data))
    return '$' + str(int(response.json()['totalBillable']) / 100)


def get_login_token_str(clockifyCredentials: dict, headers: dict) -> str:
    """
    """
    response = post('https://global.api.clockify.me/auth/token', headers=headers, data=dumps(clockifyCredentials))
    return response.json()['token']


def get_report_pdf_file(data: dict, headers: dict, workspaceStr: str) -> bytes:
    """
    """
    params = (
        ('export', 'pdf'),
    )
    response = post('https://global.api.clockify.me/workspaces/{}/reports/summary'.format(workspaceStr),
                    headers=headers, params=params, data=dumps(data))
    return response.content


def get_workspace_str(headers: dict) -> str:
    """
    """
    response = get('https://global.api.clockify.me/workspaces/', headers=headers)
    return response.json()[0]['memberships'][0]['targetId']  # Assume the first workspace and first membership...


def main(configDict) -> None:
    """
    The logic of the file.
    """
    yesterday = datetime.utcnow() - timedelta(days=1)
    yesterdayDateStr = yesterday.strftime('%Y-%m-%d')
    lastWeekDateStr = yesterday - timedelta(days=6)
    lastWeekDateStr = lastWeekDateStr.strftime('%Y-%m-%d')
    configDict['data']['startDate'] = configDict['data']['startDate'].format(lastWeekDateStr)
    configDict['data']['endDate'] = configDict['data']['endDate'].format(yesterdayDateStr)
    clockifyCredentials = configDict['clockifyCredentials']
    headers = configDict['headers']
    headers['X-Auth-Token'] = get_login_token_str(clockifyCredentials=clockifyCredentials, headers=headers)
    workspaceStr = get_workspace_str(headers=headers)
    data = configDict['data']
    pdfFile = get_report_pdf_file(data=data, headers=headers, workspaceStr=workspaceStr)
    try:
        billStr = get_bill_str(data=data, headers=headers, workspaceStr=workspaceStr)
    except Exception as exception:
        print('Exception in getting the total bill as a string: {}'.format(exception))
        billStr = 'ERROR'
    bodyStr = configDict['bodyStr'].format(yesterdayDateStr, billStr)
    ccStr = configDict['ccStr']
    emailHostAddress = configDict['emailHostAddress']
    fromEmailPasswordStr = configDict['fromEmailPasswordStr']
    fromEmailStr = configDict['fromEmailStr']
    pdfAttachmentNameStr = configDict['pdfAttachmentNameStr'].format(yesterdayDateStr)
    portInt = configDict['portInt']
    subjectStr = configDict['subjectStr']
    toEmailsStrList = configDict['toEmailsStrList'].format(yesterdayDateStr)
    email_weekly_report(bodyStr=bodyStr, ccStr=ccStr, emailHostAddress=emailHostAddress, fromEmailStr=fromEmailStr,
                        fromEmailPasswordStr=fromEmailPasswordStr, pdfAttachmentNameStr=pdfAttachmentNameStr,
                        pdfFile=pdfFile, portInt=portInt, subjectStr=subjectStr, toEmailsStrList=toEmailsStrList)


if __name__ == '__main__':
    ARG_PARSER = ArgumentParser(description='CWRS: clockify weekly report sender')
    ARG_PARSER.add_argument('configFile', help='The JSON formatted config file. Example on GitHub.', type=FileType('r'))
    ARGS = ARG_PARSER.parse_args()
    main(configDict=load(ARGS.configFile))
