FROM mailhog/mailhog:v1.0.1

USER root
COPY start.sh /start.sh
RUN chmod +x /start.sh
USER mailhog

EXPOSE 1025
# Render injects PORT; start.sh binds the UI to it.

ENTRYPOINT ["/start.sh"]
