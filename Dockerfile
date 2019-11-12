FROM python
RUN pip install requests
COPY cwrs.py /cwrs.py
