import { useState } from "react";
import { useNavigate } from "react-router-dom";
import { useTranslation } from "react-i18next";
import { Button, Form, Input, message, notification } from "antd";
import { createSchemaFieldRule } from "antd-zod";
import { z } from "zod";

import { useAntdForm } from "@/hooks";
import { getPocketBase } from "@/repository/pocketbase";
import { getErrMsg } from "@/utils/error";

const SettingsPassword = () => {
  const navigate = useNavigate();

  const { t } = useTranslation();

  const [messageApi, MessageContextHolder] = message.useMessage();
  const [notificationApi, NotificationContextHolder] = notification.useNotification();

  const formSchema = z
    .object({
      oldPassword: z
        .string({ message: t("settings.password.form.old_password.placeholder") })
        .min(10, { message: t("settings.password.form.password.errmsg.invalid") }),
      newPassword: z
        .string({ message: t("settings.password.form.new_password.placeholder") })
        .min(10, { message: t("settings.password.form.password.errmsg.invalid") }),
      confirmPassword: z
        .string({ message: t("settings.password.form.confirm_password.placeholder") })
        .min(10, { message: t("settings.password.form.password.errmsg.invalid") }),
    })
    .refine((data) => data.newPassword === data.confirmPassword, {
      message: t("settings.password.form.password.errmsg.not_matched"),
      path: ["confirmPassword"],
    });
  const formRule = createSchemaFieldRule(formSchema);
  const {
    form: formInst,
    formPending,
    formProps,
  } = useAntdForm<z.infer<typeof formSchema>>({
    onSubmit: async (values) => {
      try {
        await getPocketBase().admins.authWithPassword(getPocketBase().authStore.model?.email, values.oldPassword);
        await getPocketBase().admins.update(getPocketBase().authStore.model?.id, {
          password: values.newPassword,
          passwordConfirm: values.confirmPassword,
        });

        messageApi.success(t("common.text.operation_succeeded"));

        setTimeout(() => {
          getPocketBase().authStore.clear();
          navigate("/login");
        }, 500);
      } catch (err) {
        notificationApi.error({ message: t("common.text.request_error"), description: getErrMsg(err) });
      }
    },
  });

  const [formChanged, setFormChanged] = useState(false);

  const handleInputChange = () => {
    const values = formInst.getFieldsValue();
    setFormChanged(!!values.oldPassword && !!values.newPassword && !!values.confirmPassword);
  };

  return (
    <>
      {MessageContextHolder}
      {NotificationContextHolder}

      <div className="md:max-w-[40rem]">
        <Form {...formProps} form={formInst} disabled={formPending} layout="vertical">
          <Form.Item name="oldPassword" label={t("settings.password.form.old_password.label")} rules={[formRule]}>
            <Input.Password placeholder={t("settings.password.form.old_password.placeholder")} onChange={handleInputChange} />
          </Form.Item>

          <Form.Item name="newPassword" label={t("settings.password.form.new_password.label")} rules={[formRule]}>
            <Input.Password placeholder={t("settings.password.form.new_password.placeholder")} onChange={handleInputChange} />
          </Form.Item>

          <Form.Item name="confirmPassword" label={t("settings.password.form.confirm_password.label")} rules={[formRule]}>
            <Input.Password placeholder={t("settings.password.form.confirm_password.placeholder")} onChange={handleInputChange} />
          </Form.Item>

          <Form.Item>
            <Button type="primary" htmlType="submit" disabled={!formChanged} loading={formPending}>
              {t("common.button.save")}
            </Button>
          </Form.Item>
        </Form>
      </div>
    </>
  );
};

export default SettingsPassword;
