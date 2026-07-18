# M365 Copilot image input references

## Upload flow

The enterprise image-input path is:

1. `POST https://substrate.office.com/m365Copilot/UploadFile`
2. Send the image as multipart `FileBase64` together with the upload scenario and conversation id.
3. Include the feature gate header:

   ```http
   X-Variants: feature.EnableImageSupportInUploadFile
   ```

4. Read the returned `docId`/file metadata.
5. Attach the returned document id in the ChatHub message as a `messageAnnotations` item with `messageAnnotationType: ImageFile`.

## Sources

- `cramt/m365-copilot-proxy`, commit `b2fae5681a7e3048a8662b8a6c235f79b2cac891`, `docs/hypotheses.md` sections H8.10 and E-O3.
- The same document identifies `microsoft/PyRIT`'s historical `websocket_copilot_target.py` flow as the enterprise reference.
- `kuchris/m365-copilot-openai-proxy` is useful for the browser Substrate-token capture path, but its README does not implement image upload.

## Local validation status

The Go client now sends the feature-gate header. A real live result must still be recorded separately; compilation or unit tests alone do not prove that the upstream endpoint accepts the current account/token and multipart shape.
