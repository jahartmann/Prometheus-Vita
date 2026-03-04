"use client";

import { Badge } from "@/components/ui/badge";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import type { PackageUpdate } from "@/types/api";

interface PackageListProps {
  packages: PackageUpdate[];
}

export function PackageList({ packages }: PackageListProps) {
  if (packages.length === 0) {
    return (
      <p className="text-sm text-muted-foreground py-4 text-center">
        Keine verfuegbaren Updates.
      </p>
    );
  }

  return (
    <Table>
      <TableHeader>
        <TableRow>
          <TableHead>Paket</TableHead>
          <TableHead>Aktuelle Version</TableHead>
          <TableHead>Neue Version</TableHead>
          <TableHead>Typ</TableHead>
          <TableHead>Quelle</TableHead>
        </TableRow>
      </TableHeader>
      <TableBody>
        {packages.map((pkg) => (
          <TableRow key={pkg.name}>
            <TableCell className="font-medium font-mono text-sm">{pkg.name}</TableCell>
            <TableCell className="text-muted-foreground font-mono text-sm">
              {pkg.current_version}
            </TableCell>
            <TableCell className="font-mono text-sm">{pkg.new_version}</TableCell>
            <TableCell>
              {pkg.is_security ? (
                <Badge variant="default" className="bg-red-500">Sicherheit</Badge>
              ) : (
                <Badge variant="secondary">Standard</Badge>
              )}
            </TableCell>
            <TableCell className="text-muted-foreground text-sm">{pkg.source || "-"}</TableCell>
          </TableRow>
        ))}
      </TableBody>
    </Table>
  );
}
