"use client";

import { TagManager } from "@/components/tags/tag-manager";

export default function TagsSettingsPage() {
  return (
    <div className="space-y-4">
      <div>
        <h2 className="text-lg font-semibold">Tag-Verwaltung</h2>
        <p className="text-sm text-muted-foreground">Tags erstellen und verwalten.</p>
      </div>
      <TagManager />
    </div>
  );
}
