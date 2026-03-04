"use client";

import { useEffect, useState } from "react";
import {
  Plus,
  Pencil,
  Trash2,
  Send,
  MoreHorizontal,
} from "lucide-react";
import { Button } from "@/components/ui/button";
import { Card, CardContent } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import {
  Tabs,
  TabsContent,
  TabsList,
  TabsTrigger,
} from "@/components/ui/tabs";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { useNotificationStore } from "@/stores/notification-store";
import { useNodeStore } from "@/stores/node-store";
import { useEscalationStore } from "@/stores/escalation-store";
import { notificationApi, alertApi, escalationApi } from "@/lib/api";
import { ChannelFormDialog } from "@/components/notifications/channel-form-dialog";
import { AlertRuleDialog } from "@/components/notifications/alert-rule-dialog";
import { NotificationHistoryList } from "@/components/notifications/notification-history-list";
import { EscalationPolicyDialog } from "@/components/notifications/escalation-policy-dialog";
import { IncidentList } from "@/components/notifications/incident-list";
import { TelegramLinkCard } from "@/components/notifications/telegram-link-card";
import { SmtpConfigCard } from "@/components/notifications/smtp-config-card";
import type { NotificationChannel, AlertRule, AlertSeverity, EscalationPolicy } from "@/types/api";

const channelTypeBadge: Record<string, "default" | "secondary" | "outline"> = {
  email: "default",
  telegram: "secondary",
  webhook: "outline",
};

const severityVariant: Record<AlertSeverity, "default" | "secondary" | "destructive" | "outline"> = {
  info: "outline",
  warning: "secondary",
  critical: "destructive",
};

const severityLabel: Record<AlertSeverity, string> = {
  info: "Info",
  warning: "Warnung",
  critical: "Kritisch",
};

const formatDate = (dateStr?: string | null) => {
  if (!dateStr) return "-";
  return new Date(dateStr).toLocaleString("de-DE", {
    day: "2-digit",
    month: "2-digit",
    year: "numeric",
    hour: "2-digit",
    minute: "2-digit",
  });
};

export default function NotificationsSettingsPage() {
  const {
    channels,
    history,
    alertRules,
    isLoading,
    fetchChannels,
    fetchHistory,
    fetchAlertRules,
  } = useNotificationStore();

  const { nodes, fetchNodes } = useNodeStore();
  const {
    policies: escalationPolicies,
    incidents,
    isLoading: escalationLoading,
    fetchPolicies,
    fetchIncidents,
  } = useEscalationStore();

  const [channelDialogOpen, setChannelDialogOpen] = useState(false);
  const [editChannel, setEditChannel] = useState<NotificationChannel | null>(null);
  const [ruleDialogOpen, setRuleDialogOpen] = useState(false);
  const [editRule, setEditRule] = useState<AlertRule | null>(null);
  const [testingId, setTestingId] = useState<string | null>(null);
  const [policyDialogOpen, setPolicyDialogOpen] = useState(false);
  const [editPolicy, setEditPolicy] = useState<EscalationPolicy | null>(null);

  useEffect(() => {
    fetchChannels();
    fetchHistory();
    fetchAlertRules();
    fetchNodes();
    fetchPolicies();
    fetchIncidents();
  }, [fetchChannels, fetchHistory, fetchAlertRules, fetchNodes, fetchPolicies, fetchIncidents]);

  const handleDeleteChannel = async (id: string) => {
    await notificationApi.deleteChannel(id);
    fetchChannels();
  };

  const handleTestChannel = async (id: string) => {
    setTestingId(id);
    try {
      await notificationApi.testChannel(id);
    } catch {
      // Error handled silently, user sees test result in history
    } finally {
      setTestingId(null);
      fetchHistory();
    }
  };

  const handleDeleteRule = async (id: string) => {
    await alertApi.deleteRule(id);
    fetchAlertRules();
  };

  const handleToggleRule = async (rule: AlertRule) => {
    await alertApi.updateRule(rule.id, { is_active: !rule.is_active });
    fetchAlertRules();
  };

  const handleDeletePolicy = async (id: string) => {
    await escalationApi.deletePolicy(id);
    fetchPolicies();
  };

  return (
    <div className="space-y-4">
      <div>
        <h2 className="text-lg font-semibold">Benachrichtigungen</h2>
        <p className="text-sm text-muted-foreground">
          Kanaele, Alert-Regeln und Benachrichtigungsverlauf verwalten.
        </p>
      </div>

      <SmtpConfigCard channels={channels} onSaved={() => { fetchChannels(); fetchHistory(); }} />

      <TelegramLinkCard />

      <Tabs defaultValue="channels">
        <TabsList>
          <TabsTrigger value="channels">Kanaele</TabsTrigger>
          <TabsTrigger value="rules">Alert-Regeln</TabsTrigger>
          <TabsTrigger value="escalation">Eskalation</TabsTrigger>
          <TabsTrigger value="incidents">Vorfaelle</TabsTrigger>
          <TabsTrigger value="history">Verlauf</TabsTrigger>
        </TabsList>

        {/* Channels Tab */}
        <TabsContent value="channels" className="space-y-4">
          <div className="flex justify-end">
            <Button onClick={() => { setEditChannel(null); setChannelDialogOpen(true); }}>
              <Plus className="mr-2 h-4 w-4" />
              Kanal erstellen
            </Button>
          </div>

          <Card>
            <CardContent className="p-0">
              <Table>
                <TableHeader>
                  <TableRow>
                    <TableHead>Name</TableHead>
                    <TableHead>Typ</TableHead>
                    <TableHead>Status</TableHead>
                    <TableHead>Erstellt</TableHead>
                    <TableHead className="w-12"></TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {isLoading && channels.length === 0 ? (
                    <TableRow>
                      <TableCell colSpan={5} className="text-center py-8 text-muted-foreground">
                        Laden...
                      </TableCell>
                    </TableRow>
                  ) : channels.length === 0 ? (
                    <TableRow>
                      <TableCell colSpan={5} className="text-center py-8 text-muted-foreground">
                        Keine Kanaele konfiguriert.
                      </TableCell>
                    </TableRow>
                  ) : (
                    channels.map((channel) => (
                      <TableRow key={channel.id}>
                        <TableCell className="font-medium">{channel.name}</TableCell>
                        <TableCell>
                          <Badge variant={channelTypeBadge[channel.type] || "outline"}>
                            {channel.type}
                          </Badge>
                        </TableCell>
                        <TableCell>
                          <Badge variant={channel.is_active ? "default" : "secondary"}>
                            {channel.is_active ? "Aktiv" : "Inaktiv"}
                          </Badge>
                        </TableCell>
                        <TableCell className="text-muted-foreground">
                          {formatDate(channel.created_at)}
                        </TableCell>
                        <TableCell>
                          <DropdownMenu>
                            <DropdownMenuTrigger asChild>
                              <Button variant="ghost" size="icon">
                                <MoreHorizontal className="h-4 w-4" />
                              </Button>
                            </DropdownMenuTrigger>
                            <DropdownMenuContent align="end">
                              <DropdownMenuItem
                                onClick={() => handleTestChannel(channel.id)}
                                disabled={testingId === channel.id}
                              >
                                <Send className="mr-2 h-4 w-4" />
                                {testingId === channel.id ? "Teste..." : "Testen"}
                              </DropdownMenuItem>
                              <DropdownMenuItem
                                onClick={() => {
                                  setEditChannel(channel);
                                  setChannelDialogOpen(true);
                                }}
                              >
                                <Pencil className="mr-2 h-4 w-4" />
                                Bearbeiten
                              </DropdownMenuItem>
                              <DropdownMenuItem
                                onClick={() => handleDeleteChannel(channel.id)}
                                className="text-destructive"
                              >
                                <Trash2 className="mr-2 h-4 w-4" />
                                Loeschen
                              </DropdownMenuItem>
                            </DropdownMenuContent>
                          </DropdownMenu>
                        </TableCell>
                      </TableRow>
                    ))
                  )}
                </TableBody>
              </Table>
            </CardContent>
          </Card>
        </TabsContent>

        {/* Alert Rules Tab */}
        <TabsContent value="rules" className="space-y-4">
          <div className="flex justify-end">
            <Button onClick={() => { setEditRule(null); setRuleDialogOpen(true); }}>
              <Plus className="mr-2 h-4 w-4" />
              Regel erstellen
            </Button>
          </div>

          <Card>
            <CardContent className="p-0">
              <Table>
                <TableHeader>
                  <TableRow>
                    <TableHead>Name</TableHead>
                    <TableHead>Metrik</TableHead>
                    <TableHead>Bedingung</TableHead>
                    <TableHead>Schweregrad</TableHead>
                    <TableHead>Status</TableHead>
                    <TableHead>Letzter Alarm</TableHead>
                    <TableHead className="w-12"></TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {isLoading && alertRules.length === 0 ? (
                    <TableRow>
                      <TableCell colSpan={7} className="text-center py-8 text-muted-foreground">
                        Laden...
                      </TableCell>
                    </TableRow>
                  ) : alertRules.length === 0 ? (
                    <TableRow>
                      <TableCell colSpan={7} className="text-center py-8 text-muted-foreground">
                        Keine Alert-Regeln konfiguriert.
                      </TableCell>
                    </TableRow>
                  ) : (
                    alertRules.map((rule) => (
                      <TableRow key={rule.id}>
                        <TableCell className="font-medium">{rule.name}</TableCell>
                        <TableCell>{rule.metric}</TableCell>
                        <TableCell className="font-mono text-sm">
                          {rule.operator} {rule.threshold}
                        </TableCell>
                        <TableCell>
                          <Badge variant={severityVariant[rule.severity]}>
                            {severityLabel[rule.severity]}
                          </Badge>
                        </TableCell>
                        <TableCell>
                          <Badge
                            variant={rule.is_active ? "default" : "secondary"}
                            className="cursor-pointer"
                            onClick={() => handleToggleRule(rule)}
                          >
                            {rule.is_active ? "Aktiv" : "Inaktiv"}
                          </Badge>
                        </TableCell>
                        <TableCell className="text-muted-foreground">
                          {formatDate(rule.last_triggered_at)}
                        </TableCell>
                        <TableCell>
                          <DropdownMenu>
                            <DropdownMenuTrigger asChild>
                              <Button variant="ghost" size="icon">
                                <MoreHorizontal className="h-4 w-4" />
                              </Button>
                            </DropdownMenuTrigger>
                            <DropdownMenuContent align="end">
                              <DropdownMenuItem
                                onClick={() => {
                                  setEditRule(rule);
                                  setRuleDialogOpen(true);
                                }}
                              >
                                <Pencil className="mr-2 h-4 w-4" />
                                Bearbeiten
                              </DropdownMenuItem>
                              <DropdownMenuItem
                                onClick={() => handleDeleteRule(rule.id)}
                                className="text-destructive"
                              >
                                <Trash2 className="mr-2 h-4 w-4" />
                                Loeschen
                              </DropdownMenuItem>
                            </DropdownMenuContent>
                          </DropdownMenu>
                        </TableCell>
                      </TableRow>
                    ))
                  )}
                </TableBody>
              </Table>
            </CardContent>
          </Card>
        </TabsContent>

        {/* Escalation Tab */}
        <TabsContent value="escalation" className="space-y-4">
          <div className="flex justify-end">
            <Button onClick={() => { setEditPolicy(null); setPolicyDialogOpen(true); }}>
              <Plus className="mr-2 h-4 w-4" />
              Richtlinie erstellen
            </Button>
          </div>

          <Card>
            <CardContent className="p-0">
              <Table>
                <TableHeader>
                  <TableRow>
                    <TableHead>Name</TableHead>
                    <TableHead>Beschreibung</TableHead>
                    <TableHead>Stufen</TableHead>
                    <TableHead>Status</TableHead>
                    <TableHead className="w-12"></TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {escalationLoading && escalationPolicies.length === 0 ? (
                    <TableRow>
                      <TableCell colSpan={5} className="text-center py-8 text-muted-foreground">
                        Laden...
                      </TableCell>
                    </TableRow>
                  ) : escalationPolicies.length === 0 ? (
                    <TableRow>
                      <TableCell colSpan={5} className="text-center py-8 text-muted-foreground">
                        Keine Eskalationsrichtlinien konfiguriert.
                      </TableCell>
                    </TableRow>
                  ) : (
                    escalationPolicies.map((policy) => (
                      <TableRow key={policy.id}>
                        <TableCell className="font-medium">{policy.name}</TableCell>
                        <TableCell className="text-muted-foreground">
                          {policy.description || "-"}
                        </TableCell>
                        <TableCell>{policy.steps?.length || 0}</TableCell>
                        <TableCell>
                          <Badge variant={policy.is_active ? "default" : "secondary"}>
                            {policy.is_active ? "Aktiv" : "Inaktiv"}
                          </Badge>
                        </TableCell>
                        <TableCell>
                          <DropdownMenu>
                            <DropdownMenuTrigger asChild>
                              <Button variant="ghost" size="icon">
                                <MoreHorizontal className="h-4 w-4" />
                              </Button>
                            </DropdownMenuTrigger>
                            <DropdownMenuContent align="end">
                              <DropdownMenuItem
                                onClick={() => {
                                  setEditPolicy(policy);
                                  setPolicyDialogOpen(true);
                                }}
                              >
                                <Pencil className="mr-2 h-4 w-4" />
                                Bearbeiten
                              </DropdownMenuItem>
                              <DropdownMenuItem
                                onClick={() => handleDeletePolicy(policy.id)}
                                className="text-destructive"
                              >
                                <Trash2 className="mr-2 h-4 w-4" />
                                Loeschen
                              </DropdownMenuItem>
                            </DropdownMenuContent>
                          </DropdownMenu>
                        </TableCell>
                      </TableRow>
                    ))
                  )}
                </TableBody>
              </Table>
            </CardContent>
          </Card>
        </TabsContent>

        {/* Incidents Tab */}
        <TabsContent value="incidents">
          <IncidentList
            incidents={incidents}
            isLoading={escalationLoading}
            onRefresh={fetchIncidents}
          />
        </TabsContent>

        {/* History Tab */}
        <TabsContent value="history">
          <NotificationHistoryList entries={history} isLoading={isLoading} />
        </TabsContent>
      </Tabs>

      <ChannelFormDialog
        open={channelDialogOpen}
        onOpenChange={setChannelDialogOpen}
        onSuccess={() => { fetchChannels(); fetchHistory(); }}
        channel={editChannel}
      />

      <AlertRuleDialog
        open={ruleDialogOpen}
        onOpenChange={setRuleDialogOpen}
        onSuccess={fetchAlertRules}
        rule={editRule}
        nodes={nodes}
        channels={channels}
        escalationPolicies={escalationPolicies}
      />

      <EscalationPolicyDialog
        open={policyDialogOpen}
        onOpenChange={setPolicyDialogOpen}
        onSuccess={fetchPolicies}
        policy={editPolicy}
        channels={channels}
      />
    </div>
  );
}
