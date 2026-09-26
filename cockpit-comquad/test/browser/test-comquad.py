#!/usr/bin/env python3
"""
Browser integration tests for cockpit-comquad plugin.

These tests run inside a container with:
- systemd as PID 1
- podman + comquad installed
- cockpit-ws running
- chromium-headless for browser automation

Usage:
    python3 test-comquad.py [test_name]
"""

import json
import os
import subprocess
import sys
import tempfile
import time
import unittest
from pathlib import Path

# Try to import selenium, skip tests if not available
try:
    from selenium import webdriver
    from selenium.webdriver.chrome.options import Options
    from selenium.webdriver.chrome.service import Service
    from selenium.webdriver.common.by import By
    from selenium.webdriver.support.ui import WebDriverWait
    from selenium.webdriver.support import expected_conditions as EC
    HAS_SELENIUM = True
except ImportError:
    HAS_SELENIUM = False


COMPOSE_YAML = """
services:
  web:
    image: docker.io/library/nginx:alpine
    ports:
      - "8080:80"
  db:
    image: docker.io/library/postgres:16-alpine
    environment:
      POSTGRES_PASSWORD: test
"""


class CockpitBrowserTest(unittest.TestCase):
    """Base class for cockpit-comquad browser tests."""

    @classmethod
    def setUpClass(cls):
        if not HAS_SELENIUM:
            raise unittest.SkipTest("selenium not installed")

        cls.project_dir = tempfile.mkdtemp(prefix="comquad-test-")
        cls.project_name = Path(cls.project_dir).name

        # Write compose.yaml
        compose_path = Path(cls.project_dir) / "compose.yaml"
        compose_path.write_text(COMPOSE_YAML)

        # Deploy with comquad
        result = subprocess.run(
            ["comquad", "up", "--no-diff"],
            cwd=cls.project_dir,
            capture_output=True,
            text=True,
            timeout=120,
        )
        if result.returncode != 0:
            print(f"comquad up failed: {result.stderr}", file=sys.stderr)
            raise RuntimeError(f"Failed to deploy test project: {result.stderr}")

        # Set up browser
        chrome_options = Options()
        chrome_options.add_argument("--headless")
        chrome_options.add_argument("--no-sandbox")
        chrome_options.add_argument("--disable-dev-shm-usage")
        chrome_options.add_argument("--disable-gpu")

        try:
            cls.driver = webdriver.Chrome(options=chrome_options)
            cls.driver.set_window_size(1280, 1024)
        except Exception as e:
            raise unittest.SkipTest(f"Cannot start browser: {e}")

        # Login to cockpit
        cls.driver.get("http://localhost:9090")
        time.sleep(2)

        # Cockpit login
        try:
            username_input = cls.driver.find_element(By.ID, "login-user-input")
            password_input = cls.driver.find_element(By.ID, "login-password-input")
            login_button = cls.driver.find_element(By.ID, "login-button")

            username_input.send_keys("root")
            password_input.send_keys("")
            login_button.click()
            time.sleep(3)
        except Exception:
            # May already be logged in or login page structure changed
            pass

    @classmethod
    def tearDownClass(cls):
        if hasattr(cls, "driver"):
            cls.driver.quit()

        # Clean up comquad project
        subprocess.run(
            ["comquad", "down", "-y"],
            cwd=cls.project_dir,
            capture_output=True,
            timeout=60,
        )

        # Clean up temp dir
        import shutil
        shutil.rmtree(cls.project_dir, ignore_errors=True)

    def navigate_to_comquad(self):
        """Navigate to the Comquad Stacks page."""
        self.driver.get("http://localhost:9090/comquad")
        time.sleep(2)

    def wait_for_text(self, text, timeout=10):
        """Wait for text to appear on the page."""
        WebDriverWait(self.driver, timeout).until(
            EC.presence_of_element_located(
                (By.XPATH, f"//*[contains(text(), '{text}')]")
            )
        )

    def wait_for_element(self, selector, timeout=10):
        """Wait for an element to be present."""
        return WebDriverWait(self.driver, timeout).until(
            EC.presence_of_element_located((By.CSS_SELECTOR, selector))
        )


@unittest.skipUnless(HAS_SELENIUM, "selenium not installed")
class TestStackDashboard(CockpitBrowserTest):
    """Test the stack dashboard (projects list)."""

    def test_page_loads(self):
        """Test that the Comquad Stacks page loads."""
        self.navigate_to_comquad()
        self.wait_for_text("Comquad Stacks")

    def test_project_listed(self):
        """Test that deployed project appears in the list."""
        self.navigate_to_comquad()
        self.wait_for_text("Comquad Stacks")
        # The project name should appear in the table
        self.wait_for_text(self.project_name, timeout=15)

    def test_project_has_status(self):
        """Test that project shows a status badge."""
        self.navigate_to_comquad()
        self.wait_for_text(self.project_name, timeout=15)
        # Should show running or healthy status
        page_source = self.driver.page_source
        has_status = any(
            status in page_source.lower()
            for status in ["running", "healthy", "up"]
        )
        self.assertTrue(has_status, "No status badge found for project")

    def test_refresh_button(self):
        """Test that refresh button exists and is clickable."""
        self.navigate_to_comquad()
        self.wait_for_text("Comquad Stacks")
        refresh_btn = self.wait_for_element("button[aria-label='Refresh']")
        self.assertTrue(refresh_btn.is_displayed())
        refresh_btn.click()
        time.sleep(1)

    def test_deploy_button(self):
        """Test that Deploy Stack button exists."""
        self.navigate_to_comquad()
        self.wait_for_text("Comquad Stacks")
        self.wait_for_text("Deploy Stack")


@unittest.skipUnless(HAS_SELENIUM, "selenium not installed")
class TestProjectDetail(CockpitBrowserTest):
    """Test the project detail view."""

    def setUp(self):
        self.navigate_to_comquad()
        self.wait_for_text(self.project_name, timeout=15)
        # Click on the project to open detail view
        project_link = self.driver.find_element(
            By.XPATH, f"//*[contains(text(), '{self.project_name}')]"
        )
        project_link.click()
        time.sleep(2)

    def test_detail_view_loads(self):
        """Test that project detail view loads."""
        self.wait_for_text(self.project_name)

    def test_services_tab(self):
        """Test that Services tab shows services."""
        self.wait_for_text("Services")
        services_tab = self.driver.find_element(
            By.XPATH, "//button[contains(text(), 'Services')]"
        )
        services_tab.click()
        time.sleep(1)
        # Should show web and db services
        self.wait_for_text("web")
        self.wait_for_text("db")

    def test_containers_tab(self):
        """Test that Containers tab shows containers."""
        self.wait_for_text("Containers")
        containers_tab = self.driver.find_element(
            By.XPATH, "//button[contains(text(), 'Containers')]"
        )
        containers_tab.click()
        time.sleep(1)
        # Should show containers
        self.wait_for_text("web")

    def test_resources_tab(self):
        """Test that Resources tab shows resources."""
        self.wait_for_text("Resources")
        resources_tab = self.driver.find_element(
            By.XPATH, "//button[contains(text(), 'Resources')]"
        )
        resources_tab.click()
        time.sleep(1)
        # Should show network resource
        self.wait_for_text("network")

    def test_action_buttons(self):
        """Test that action buttons are present."""
        self.wait_for_text("Update")
        self.wait_for_text("Stop")
        self.wait_for_text("Remove")

    def test_back_button(self):
        """Test that back button works."""
        back_btn = self.wait_for_element("button[aria-label='Back']")
        back_btn.click()
        time.sleep(1)
        self.wait_for_text("Comquad Stacks")


if __name__ == "__main__":
    # Run specific test if provided as argument
    if len(sys.argv) > 1:
        test_name = sys.argv[1]
        suite = unittest.TestSuite()
        for test_class in [TestStackDashboard, TestProjectDetail]:
            for method_name in dir(test_class):
                if method_name.startswith("test_") and method_name == test_name:
                    suite.addTest(test_class(method_name))
        runner = unittest.TextTestRunner(verbosity=2)
        result = runner.run(suite)
        sys.exit(0 if result.wasSuccessful() else 1)
    else:
        unittest.main(verbosity=2)
