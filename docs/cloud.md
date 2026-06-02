Чтобы развернуть приложение из Docker-файла на Cloud.ru, выполните следующие шаги:

1. **Войдите в личный кабинет Cloud.ru**
   Если вы ещё не зарегистрированы — зарегистрируйтесь. Если уже есть аккаунт — войдите под ним.

2. **Создайте реестр в Artifact Registry**
   - Перейдите в сервис [Artifact Registry](https://console.cloud.ru/spa/artifact-registry/registries).
   - Нажмите «Создать реестр».
   - Укажите название реестра (оно станет частью адреса).
   - Выберите тип доступа — приватный.
   - Нажмите «Создать».

3. **Получите персональные ключи доступа**
   - В личном кабинете перейдите в раздел «Управление профилем» → «Ключи доступа».
   - Нажмите «Создать ключ».
   - Укажите описание и срок жизни (от 1 до 365 дней).
   - Скопируйте и сохраните **Key ID** и **Key Secret** (после закрытия окна Key Secret больше нельзя будет посмотреть).

4. **Пройдите аутентификацию в терминале**
   Откройте терминал на вашем компьютере и выполните команду:
   ```
   docker login <registry_name>.cr.cloud.ru -u <key_id> -p <key_secret>
   ```
   где:
   - `<registry_name>` — название вашего реестра,
   - `<key_id>` — логин ключа,
   - `<key_secret>` — пароль ключа.

5. **Соберите и загрузите образ**
   В папке с вашим Dockerfile выполните:
   ```
   docker build --tag <registry_name>.cr.cloud.ru/<repository_name> . --platform linux/amd64
   ```
   Затем загрузите образ:
   ```
   docker push <registry_name>.cr.cloud.ru/<repository_name>
   ```

6. **Разверните приложение в Container Apps**
   - Перейдите в [Container Apps](https://console.cloud.ru/spa/container-apps/apps).
   - Нажмите «Создать приложение».
   - Выберите образ из вашего реестра.
   - Укажите порт, конфигурацию (vCPU/RAM), количество экземпляров и включите публичный доступ, если нужно.
   - Нажмите «Создать».

После этого ваше приложение будет доступно по публичному URL.

Подробнее:
- [Быстрый старт Artifact Registry](https://cloud.ru/docs/artifact-registry-evolution/ug/topics/quickstart)
- [Быстрый старт Container Apps](https://cloud.ru/docs/container-apps-evolution/ug/topics/quickstart)
- [Подготовка среды](https://cloud.ru/docs/tutorials-evolution/list/topics/container-apps__before-work)