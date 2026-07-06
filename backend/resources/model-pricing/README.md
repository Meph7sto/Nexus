# Model Pricing Data

This directory contains the bundled model pricing data used as the default fallback.

## Purpose

The local copy lets Nexus start and calculate usage costs without depending on a remote pricing source.

## Manual Update

Replace `model_prices_and_context_window.json` with a reviewed pricing dataset that matches the same JSON format.

## File Format

The file contains JSON data with model pricing information including:
- Model names and identifiers
- Input/output token costs
- Context window sizes
- Model capabilities

Last updated: 2025-08-10
