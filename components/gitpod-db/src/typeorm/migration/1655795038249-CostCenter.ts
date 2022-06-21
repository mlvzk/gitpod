/**
 * Copyright (c) 2022 Gitpod GmbH. All rights reserved.
 * Licensed under the GNU Affero General Public License (AGPL).
 * See License-AGPL.txt in the project root for license information.
 */

import { MigrationInterface, QueryRunner } from "typeorm";
import { columnExists } from "./helper/helper";

export class CostCenter1655795038249 implements MigrationInterface {
    public async up(queryRunner: QueryRunner): Promise<void> {
        await queryRunner.query(
            "CREATE TABLE IF NOT EXISTS `d_b_cost_center` (`id` char(36) NOT NULL, `deleted` tinyint(4) NOT NULL DEFAULT '0', `_lastModified` timestamp(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6) ON UPDATE CURRENT_TIMESTAMP(6), PRIMARY KEY (`id`), KEY `ind_dbsync` (`_lastModified`)) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;",
        );
        if (!(await columnExists(queryRunner, "d_b_workspace_instance", "costCenterId"))) {
            await queryRunner.query(
                "ALTER TABLE `d_b_workspace_instance` CHANGE `attributedTeamId` `costCenterId` char(36) NOT NULL DEFAULT '', ALGORITHM=INPLACE, LOCK=NONE",
            );
        }
    }

    public async down(queryRunner: QueryRunner): Promise<void> {
        if (!(await columnExists(queryRunner, "d_b_workspace_instance", "attributedTeamId"))) {
            await queryRunner.query(
                "ALTER TABLE `d_b_workspace_instance` CHANGE `costCenterId` `attributedTeamId` char(36) NOT NULL DEFAULT '', ALGORITHM=INPLACE, LOCK=NONE",
            );
        }
    }
}
