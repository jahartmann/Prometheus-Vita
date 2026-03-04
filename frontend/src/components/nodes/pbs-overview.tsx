"use client";

import { useEffect, useState } from "react";
import { Database, Calendar } from "lucide-react";
import { Card, CardContent } from "@/components/ui/card";
import { formatBytes, formatPercentage, getUsageBgColor } from "@/lib/utils";
import { pbsApi, toArray } from "@/lib/api";
import type { PBSDatastore, PBSBackupJob } from "@/types/api";

interface PBSOverviewProps {
  nodeId: string;
}

export function PBSOverview({ nodeId }: PBSOverviewProps) {
  const [datastores, setDatastores] = useState<PBSDatastore[]>([]);
  const [jobs, setJobs] = useState<PBSBackupJob[]>([]);

  useEffect(() => {
    pbsApi
      .getDatastores(nodeId)
      .then((res) => {
        setDatastores(toArray<PBSDatastore>(res.data));
      })
      .catch(() => {});
    pbsApi
      .getBackupJobs(nodeId)
      .then((res) => {
        setJobs(toArray<PBSBackupJob>(res.data));
      })
      .catch(() => {});
  }, [nodeId]);

  return (
    <div className="space-y-6">
      <div>
        <h3 className="text-lg font-semibold mb-3">Datastores</h3>
        <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
          {datastores.map((ds) => (
            <Card key={ds.name}>
              <CardContent className="p-4">
                <div className="flex items-center gap-2 mb-3">
                  <Database className="h-5 w-5 text-primary" />
                  <span className="font-medium">{ds.name}</span>
                </div>
                {ds.total != null && ds.total > 0 ? (
                  <div className="space-y-2">
                    <div className="flex justify-between text-xs text-muted-foreground">
                      <span>{formatBytes(ds.used || 0)}</span>
                      <span>{formatBytes(ds.total)}</span>
                    </div>
                    <div className="h-2 rounded-full bg-secondary">
                      <div
                        className={`h-2 rounded-full ${getUsageBgColor(ds.usage_percent || 0)}`}
                        style={{ width: `${Math.min(ds.usage_percent || 0, 100)}%` }}
                      />
                    </div>
                    <p className="text-xs text-muted-foreground">
                      {formatPercentage(ds.usage_percent || 0)} belegt
                      {ds.gc_status && ` | GC: ${ds.gc_status}`}
                    </p>
                  </div>
                ) : (
                  <p className="text-xs text-muted-foreground">{ds.path || "Keine Daten"}</p>
                )}
              </CardContent>
            </Card>
          ))}
          {datastores.length === 0 && (
            <p className="text-sm text-muted-foreground col-span-full text-center py-4">
              Keine Datastores gefunden.
            </p>
          )}
        </div>
      </div>

      {jobs.length > 0 && (
        <div>
          <h3 className="text-lg font-semibold mb-3">Backup Jobs</h3>
          <div className="space-y-2">
            {jobs.map((job) => (
              <Card key={job.id}>
                <CardContent className="flex items-center justify-between p-4">
                  <div className="flex items-center gap-3">
                    <Calendar className="h-4 w-4 text-muted-foreground" />
                    <div>
                      <p className="font-medium">{job.id}</p>
                      <p className="text-xs text-muted-foreground">
                        Store: {job.store}
                        {job.schedule && ` | ${job.schedule}`}
                        {job.remote && ` | Remote: ${job.remote}`}
                      </p>
                    </div>
                  </div>
                </CardContent>
              </Card>
            ))}
          </div>
        </div>
      )}
    </div>
  );
}
