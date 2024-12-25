import { useTranslation } from "react-i18next";
import { Form, Input, InputNumber, Switch } from "antd";
import { createSchemaFieldRule } from "antd-zod";
import { z } from "zod";

const NotifyChannelEditFormEmailFields = () => {
  const { t } = useTranslation();

  const formSchema = z.object({
    smtpHost: z
      .string({ message: t("settings.notification.channel.form.email_smtp_host.placeholder") })
      .min(1, t("settings.notification.channel.form.email_smtp_host.placeholder"))
      .max(256, t("common.errmsg.string_max", { max: 256 })),
    smtpPort: z
      .number({ message: t("settings.notification.channel.form.email_smtp_port.placeholder") })
      .int()
      .gte(1, t("common.errmsg.port_invalid"))
      .lte(65535, t("common.errmsg.port_invalid"))
      .transform((v) => +v),
    smtpTLS: z.boolean().nullish(),
    username: z
      .string({ message: t("settings.notification.channel.form.email_username.placeholder") })
      .min(1, t("settings.notification.channel.form.email_username.placeholder"))
      .max(256, t("common.errmsg.string_max", { max: 256 })),
    password: z
      .string({ message: t("settings.notification.channel.form.email_password.placeholder") })
      .min(1, t("settings.notification.channel.form.email_password.placeholder"))
      .max(256, t("common.errmsg.string_max", { max: 256 })),
    senderAddress: z
      .string({ message: t("settings.notification.channel.form.email_sender_address.placeholder") })
      .min(1, t("settings.notification.channel.form.email_sender_address.placeholder"))
      .email({ message: t("common.errmsg.email_invalid") }),
    receiverAddress: z
      .string({ message: t("settings.notification.channel.form.email_receiver_address.placeholder") })
      .min(1, t("settings.notification.channel.form.email_receiver_address.placeholder"))
      .email({ message: t("common.errmsg.email_invalid") }),
  });
  const formRule = createSchemaFieldRule(formSchema);
  const form = Form.useFormInstance<z.infer<typeof formSchema>>();

  const handleTLSSwitchChange = (checked: boolean) => {
    const oldPort = form.getFieldValue("smtpPort");
    const newPort = checked && (oldPort == null || oldPort === 25) ? 465 : !checked && (oldPort == null || oldPort === 465) ? 25 : oldPort;
    if (newPort !== oldPort) {
      form.setFieldValue("smtpPort", newPort);
    }
  };

  return (
    <>
      <div className="flex space-x-2">
        <div className="w-2/5">
          <Form.Item name="smtpHost" label={t("settings.notification.channel.form.email_smtp_host.label")} rules={[formRule]}>
            <Input placeholder={t("settings.notification.channel.form.email_smtp_host.placeholder")} />
          </Form.Item>
        </div>

        <div className="w-2/5">
          <Form.Item name="smtpPort" label={t("settings.notification.channel.form.email_smtp_port.label")} rules={[formRule]}>
            <InputNumber className="w-full" placeholder={t("settings.notification.channel.form.email_smtp_port.placeholder")} min={1} max={65535} />
          </Form.Item>
        </div>

        <div className="w-1/5">
          <Form.Item name="smtpTLS" label={t("settings.notification.channel.form.email_smtp_tls.label")} rules={[formRule]}>
            <Switch onChange={handleTLSSwitchChange} />
          </Form.Item>
        </div>
      </div>

      <div className="flex space-x-2">
        <div className="w-1/2">
          <Form.Item name="username" label={t("settings.notification.channel.form.email_username.label")} rules={[formRule]}>
            <Input placeholder={t("settings.notification.channel.form.email_username.placeholder")} />
          </Form.Item>
        </div>

        <div className="w-1/2">
          <Form.Item name="password" label={t("settings.notification.channel.form.email_password.label")} rules={[formRule]}>
            <Input.Password autoComplete="new-password" placeholder={t("settings.notification.channel.form.email_password.placeholder")} />
          </Form.Item>
        </div>
      </div>

      <Form.Item name="senderAddress" label={t("settings.notification.channel.form.email_sender_address.label")} rules={[formRule]}>
        <Input type="email" placeholder={t("settings.notification.channel.form.email_sender_address.placeholder")} />
      </Form.Item>

      <Form.Item name="receiverAddress" label={t("settings.notification.channel.form.email_receiver_address.label")} rules={[formRule]}>
        <Input type="email" placeholder={t("settings.notification.channel.form.email_receiver_address.placeholder")} />
      </Form.Item>
    </>
  );
};

export default NotifyChannelEditFormEmailFields;
