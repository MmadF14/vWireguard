install.sh
        certbot --nginx --non-interactive --agree-tos -m "$LE_EMAIL" -d "$PANEL_DOMAIN"
    else
        certbot --nginx --register-unsafely-without-email --non-interactive --agree-tos -d "$PANEL_DOMAIN"
    fi
fi

# Create default admin user
echo -e "${YELLOW}Creating default admin user...${NC}"
>>>>>>> parent of 37fbd02 (Add optional SSL setup)


@@ -241,6 +218,7 @@ echo -e "\n${YELLOW}=======================================================${NC}
echo -e "${GREEN}Default Admin Credentials:${NC}"
echo -e "  ${YELLOW}Username: admin${NC}"
echo -e "  ${YELLOW}Password: admin${NC}"

echo -e "${YELLOW}=======================================================${NC}\n"
>>>>>>> parent of 37fbd02 (Add optional SSL setup)
=======
echo -e "${GREEN}Access URL: http://$(curl -s ifconfig.me):5000${NC}"
echo -e "${YELLOW}=======================================================${NC}\n"
>>>>>>> parent of 37fbd02 (Add optional SSL setup)