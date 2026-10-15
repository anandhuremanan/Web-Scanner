import { ImageResponse } from "next/og";

// The picture shown when a link to the site is shared (chats, social posts).
export const alt = "Web Scanner: find every subdomain and URL of a website";
export const size = { width: 1200, height: 630 };
export const contentType = "image/png";

export default function OpengraphImage() {
  return new ImageResponse(
    (
      <div
        style={{
          width: "100%",
          height: "100%",
          display: "flex",
          flexDirection: "column",
          justifyContent: "center",
          padding: "80px",
          background: "#111110",
          color: "#ececea",
          fontFamily: "sans-serif",
        }}
      >
        <div style={{ display: "flex", alignItems: "center", gap: "24px" }}>
          <div
            style={{
              width: "72px",
              height: "72px",
              borderRadius: "36px",
              border: "8px solid #7d9df0",
              display: "flex",
              alignItems: "center",
              justifyContent: "center",
            }}
          >
            <div style={{ width: "16px", height: "16px", borderRadius: "8px", background: "#ececea" }} />
          </div>
          <div style={{ fontSize: "64px", fontWeight: 700 }}>Web Scanner</div>
        </div>
        <div style={{ marginTop: "40px", fontSize: "44px", lineHeight: 1.25, maxWidth: "980px" }}>
          Find every subdomain and URL of a website, from public sources.
        </div>
        <div style={{ marginTop: "32px", fontSize: "28px", color: "#9a9a94" }}>
          Certificate logs · Wayback Machine · DNS · Sitemaps · Free web app and CLI
        </div>
      </div>
    ),
    size,
  );
}
