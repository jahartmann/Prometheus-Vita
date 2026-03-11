"use client";

import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { SystemProcesses } from "./system-processes";
import { SystemServices } from "./system-services";
import { SystemPorts } from "./system-ports";
import { SystemDisk } from "./system-disk";

export function SystemTab() {
  return (
    <Tabs defaultValue="processes">
      <TabsList>
        <TabsTrigger value="processes">Prozesse</TabsTrigger>
        <TabsTrigger value="services">Services</TabsTrigger>
        <TabsTrigger value="ports">Ports</TabsTrigger>
        <TabsTrigger value="disk">Speicher</TabsTrigger>
      </TabsList>

      <TabsContent value="processes" className="mt-4">
        <SystemProcesses />
      </TabsContent>

      <TabsContent value="services" className="mt-4">
        <SystemServices />
      </TabsContent>

      <TabsContent value="ports" className="mt-4">
        <SystemPorts />
      </TabsContent>

      <TabsContent value="disk" className="mt-4">
        <SystemDisk />
      </TabsContent>
    </Tabs>
  );
}
